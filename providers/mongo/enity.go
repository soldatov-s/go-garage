package mongo

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/base"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/sync/errgroup"
)

const ProviderName = "mongo"

var (
	ErrFailedReadInputFile = errors.New("could not read the input file")
	ErrWriteToGridFS       = errors.New("could not write to GridFs")
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.Enity
	*base.MetricsStorage
	*base.ReadyCheckStorage
	conn   *mongo.Client
	config *Config
	dbName string
	bucket *gridfs.Bucket
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, name string, config *Config) (*Enity, error) {
	deps := &base.EnityDeps{
		ProviderName: ProviderName,
		Name:         name,
	}
	baseEnity := base.NewEnity(deps)

	if config == nil {
		return nil, base.ErrInvalidEnityOptions
	}

	enity := &Enity{Enity: baseEnity, config: config.SetDefault()}
	if err := enity.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	if err := enity.buildReadyHandlers(ctx); err != nil {
		return nil, errors.Wrap(err, "build ready handlers")
	}

	return enity, nil
}

func (e *Enity) GetConn() *mongo.Client {
	return e.conn
}

func (e *Enity) GetConfig() *Config {
	return e.config
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (e *Enity) Shutdown(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("shutting down")
	e.SetShuttingDown(true)

	if e.config.StartWatcher {
		for {
			if e.IsWatcherStopped() {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	} else if err := e.shutdown(ctx); err != nil {
		return errors.Wrapf(err, "shutdown %q", e.GetFullName())
	}

	e.GetLogger(ctx).Info().Msg("shutted down")
	return nil
}

// Start starts connection workers and connection procedure itself.
func (e *Enity) Start(ctx context.Context, errorGroup *errgroup.Group) error {
	// If connection is nil - try to establish
	// connection.
	if e.conn == nil {
		e.GetLogger(ctx).Info().Msg("establishing connection to database...")
		// Connect to database.
		cs, err := connstring.ParseAndValidate(e.config.ComposeDSN())
		if err != nil {
			return errors.Wrap(err, "parse dsn")
		}
		e.dbName = cs.Database

		e.conn, err = mongo.Connect(ctx, options.Client().ApplyURI(e.config.ComposeDSN()))
		if err != nil {
			return errors.Wrap(err, "connect to enity")
		}
		e.GetLogger(ctx).Info().Msg("database connection established")
	}

	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if e.IsWatcherStopped() {
		e.SetWatcher(false)
		errorGroup.Go(func() error {
			return e.startWatcher(ctx)
		})
	}

	return nil
}

// Connection watcher goroutine entrypoint.
func (e *Enity) startWatcher(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("starting connection watcher")

	for {
		select {
		case <-ctx.Done():
			e.GetLogger(ctx).Info().Msg("connection watcher stopped")
			e.SetWatcher(true)
			return ctx.Err()
		default:
			if err := e.Ping(ctx); err != nil {
				e.GetLogger(ctx).Error().Err(err).Msg("connection lost")
			}
		}
		time.Sleep(e.config.Timeout)
	}
}

func (e *Enity) shutdown(ctx context.Context) error {
	if e.conn == nil {
		return nil
	}
	e.GetLogger(ctx).Info().Msg("closing connection...")

	if err := e.conn.Disconnect(ctx); err != nil {
		return errors.Wrap(err, "failed to close connection")
	}

	e.conn = nil

	return nil
}

// Pinging connection if it's alive (or we think so).
func (e *Enity) Ping(ctx context.Context) error {
	if e.conn == nil {
		return nil
	}

	if err := e.conn.Ping(ctx, nil); err != nil {
		return errors.Wrap(err, "ping connection")
	}

	return nil
}

// GetMetrics return map of the metrics from database connection
func (e *Enity) buildMetrics(_ context.Context) error {
	fullName := e.GetFullName()
	redactedDSN, err := stringsx.RedactedDSN(e.config.DSN)
	if err != nil {
		return errors.Wrap(err, "redacted dsn")
	}
	help := stringsx.JoinStrings(" ", "status link to", redactedDSN)
	metricFunc := func(ctx context.Context) (float64, error) {
		if e.conn != nil {
			err := e.Ping(ctx)
			if err == nil {
				return 1, nil
			}
		}
		return 0, nil
	}
	if _, err := e.MetricsStorage.GetMetrics().AddMetricGauge(fullName, "status", help, metricFunc); err != nil {
		return errors.Wrap(err, "add gauge metric")
	}

	return nil
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (e *Enity) buildReadyHandlers(_ context.Context) error {
	checkOptions := &base.CheckOptions{
		Name: strings.ToUpper(e.GetFullName() + "_notfailed"),
		CheckFunc: func(ctx context.Context) error {
			if e.conn == nil {
				return base.ErrNotConnected
			}

			if err := e.Ping(ctx); err != nil {
				return errors.Wrap(err, "ping")
			}

			return nil
		},
	}
	if err := e.ReadyCheckStorage.GetReadyHandlers().Add(checkOptions); err != nil {
		return errors.Wrap(err, "add ready handler")
	}
	return nil
}

func (e *Enity) newBucket() (*gridfs.Bucket, error) {
	if e.bucket != nil {
		return e.bucket, nil
	}

	bucket, err := gridfs.NewBucket(e.conn.Database(e.dbName))
	if err != nil {
		return nil, err
	}

	e.bucket = bucket

	return e.bucket, nil
}

func (e *Enity) writeToGridFile(
	ctx context.Context,
	fileName string,
	file multipart.File,
	gridFile *gridfs.UploadStream) (int, error) {
	reader := bufio.NewReader(file)
	defer func() { file.Close() }()
	// make a buffer to keep chunks that are read
	buf := make([]byte, 1024)
	fileSize := 0
	for {
		// read a chunk
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return 0, ErrFailedReadInputFile
		}
		if n == 0 {
			break
		}
		// write a chunk
		size, err := gridFile.Write(buf[:n])
		if err != nil {
			e.GetLogger(ctx).Err(err).Msgf("failed write to GridFS file %s", fileName)
			return 0, errors.Wrapf(ErrWriteToGridFS, "file %s", fileName)
		}
		fileSize += size
	}
	gridFile.Close()
	return fileSize, nil
}

// ObjectIDFileName is a map between objectID and file name
type ObjectIDFileName map[string]string

func (e *Enity) WriteMultipart(ctx context.Context, fileprefix string, multipartForm *multipart.Form) (ObjectIDFileName, error) {
	result := make(ObjectIDFileName)
	for _, fileHeaders := range multipartForm.File {
		for _, fileHeader := range fileHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				return nil, err
			}

			bucket, err := e.newBucket()
			if err != nil {
				return nil, err
			}

			// this is the name of the file which will be saved in the database
			filename := fileHeader.Filename
			if fileprefix != "" {
				filename = fileprefix + "_" + filename
			}

			gridFile, err := bucket.OpenUploadStream(filename)
			if err != nil {
				return nil, err
			}

			fileSize, err := e.writeToGridFile(ctx, fileHeader.Filename, file, gridFile)
			if err != nil {
				return nil, err
			}

			e.GetLogger(ctx).Debug().Msgf("write file to DB was successful; file size: %d \n", fileSize)

			result[gridFile.FileID.(primitive.ObjectID).Hex()] = filename
		}
	}

	return result, nil
}

func (e *Enity) GetFile(ctx context.Context, fileID string) (*bytes.Buffer, int64, error) {
	bucket, err := e.newBucket()
	if err != nil {
		return nil, 0, err
	}

	var buf bytes.Buffer

	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return nil, 0, err
	}

	dStream, err := bucket.DownloadToStream(objectID, &buf)
	if err != nil {
		return nil, 0, err
	}

	e.GetLogger(ctx).Debug().Msgf("file size to download: %v\n", dStream)
	return &buf, dStream, nil
}

func (e *Enity) GetFileByName(ctx context.Context, fileName, fileprefix string) (*bytes.Buffer, int64, error) {
	bucket, err := e.newBucket()
	if err != nil {
		return nil, 0, err
	}

	var buf bytes.Buffer
	// this is the name of the file which will be saved in the database
	filename := fileName
	if fileprefix != "" {
		filename = fileprefix + "_" + filename
	}

	dStream, err := bucket.DownloadToStreamByName(filename, &buf)
	if err != nil {
		return nil, 0, err
	}

	e.GetLogger(ctx).Debug().Msgf("file size to download: %v\n", dStream)
	return &buf, dStream, nil
}

func (e *Enity) DeleteFile(fileID string) error {
	bucket, err := e.newBucket()
	if err != nil {
		return err
	}

	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return err
	}

	return bucket.Delete(objectID)
}

func (e *Enity) RenameFile(fileID, newFilename string) error {
	bucket, err := e.newBucket()
	if err != nil {
		return err
	}

	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return err
	}

	return bucket.Rename(objectID, newFilename)
}

func (e *Enity) UpdateFile(ctx context.Context, fileID, fileprefix string, multipartForm *multipart.Form) error {
	bucket, err := e.newBucket()
	if err != nil {
		return err
	}

	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return err
	}

	err = bucket.Delete(objectID)
	if err != nil {
		return err
	}

	for _, fileHeaders := range multipartForm.File {
		for _, fileHeader := range fileHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				return err
			}

			bucket, err := e.newBucket()
			if err != nil {
				return err
			}

			// this is the name of the file which will be saved in the database
			filename := fileHeader.Filename
			if fileprefix != "" {
				filename = fileprefix + "_" + filename
			}

			gridFile, err := bucket.OpenUploadStreamWithID(objectID, filename)
			if err != nil {
				return err
			}

			fileSize, err := e.writeToGridFile(ctx, fileHeader.Filename, file, gridFile)
			if err != nil {
				return err
			}

			e.GetLogger(ctx).Debug().Msgf("write file to DB was successful; file size: %d \n", fileSize)
		}
	}

	return nil
}
