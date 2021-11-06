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
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/soldatov-s/go-garage/x/helper"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/sync/errgroup"
)

var (
	ErrFailedReadInputFile = errors.New("could not read the input file")
	ErrWriteToGridFS       = errors.New("could not write to GridFs")
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.EntityWithMetrics
	Conn   *mongo.Client
	cfg    *Config
	dbName string
	bucket *gridfs.Bucket
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, collectorName, providerName, name string, cfg interface{}) (*Enity, error) {
	e, err := base.NewEntityWithMetrics(ctx, collectorName, providerName, name)
	if err != nil {
		return nil, errors.Wrap(err, "create base enity")
	}

	// Checking that passed config is OUR.
	config, ok := cfg.(*Config)
	if !ok {
		return nil, errors.Wrapf(base.ErrInvalidEnityOptions, "expected %q", helper.ObjName(Config{}))
	}

	return &Enity{EntityWithMetrics: e, cfg: config.SetDefault()}, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (e *Enity) Shutdown(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("shutting down")
	e.WeAreShuttingDown = true

	if e.cfg.StartWatcher {
		for {
			if e.ConnWatcherStopped {
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
	if e.Conn == nil {
		e.GetLogger(ctx).Info().Msg("establishing connection to database...")
		// Connect to database.
		cs, err := connstring.ParseAndValidate(e.cfg.ComposeDSN())
		if err != nil {
			return errors.Wrap(err, "parse dsn")
		}
		e.dbName = cs.Database

		e.Conn, err = mongo.Connect(ctx, options.Client().ApplyURI(e.cfg.ComposeDSN()))
		if err != nil {
			return errors.Wrap(err, "connect to enity")
		}
		e.GetLogger(ctx).Info().Msg("database connection established")
	}

	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if e.ConnWatcherStopped {
		e.ConnWatcherStopped = false
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
			e.ConnWatcherStopped = true
			return ctx.Err()
		default:
			if err := e.Ping(ctx); err != nil {
				e.GetLogger(ctx).Error().Err(err).Msg("connection lost")
			}
		}
		time.Sleep(e.cfg.Timeout)
	}
}

func (e *Enity) shutdown(ctx context.Context) error {
	if e.Conn == nil {
		return nil
	}
	e.GetLogger(ctx).Info().Msg("closing connection...")

	err := e.Conn.Disconnect(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to close connection")
	}

	e.Conn = nil

	return nil
}

// Pinging connection if it's alive (or we think so).
func (e *Enity) Ping(ctx context.Context) error {
	if e.Conn == nil {
		return nil
	}

	if err := e.Conn.Ping(ctx, nil); err != nil {
		return errors.Wrap(err, "ping connection")
	}

	return nil
}

// GetMetrics return map of the metrics from database connection
func (e *Enity) GetMetrics(ctx context.Context) base.MapMetricsOptions {
	e.Metrics.AddNewMetricGauge(
		e.GetFullName(),
		"status",
		utils.JoinStrings(" ", "status link to", utils.RedactedDSN(e.cfg.DSN)),
		func() float64 {
			if e.Conn != nil {
				err := e.Ping(ctx)
				if err == nil {
					return 1
				}
			}
			return 0
		},
	)

	return e.Metrics
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (e *Enity) GetReadyHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	e.ReadyHandlers[strings.ToUpper(e.GetFullName()+"_notfailed")] = func() (bool, string) {
		if e.Conn == nil {
			return false, "not connected"
		}

		if err := e.Ping(ctx); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return e.ReadyHandlers, nil
}

func (e *Enity) newBucket() (*gridfs.Bucket, error) {
	if e.bucket != nil {
		return e.bucket, nil
	}

	bucket, err := gridfs.NewBucket(e.Conn.Database(e.dbName))
	if err != nil {
		return nil, err
	}

	e.bucket = bucket

	return e.bucket, nil
}

func (e *Enity) writeToGridFile(ctx context.Context, fileName string, file multipart.File, gridFile *gridfs.UploadStream) (int, error) {
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
