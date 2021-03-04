package mongo

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	// Metrics
	stats.Service

	// Connection to database
	Conn   *mongo.Client
	ctx    context.Context
	log    zerolog.Logger
	dbName string
	bucket *gridfs.Bucket
	name   string
	cfg    *Config

	// Shutdown flags.
	weAreShuttingDown  bool
	connWatcherStopped bool
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, name string, cfg interface{}) (*Enity, error) {
	if name == "" {
		return nil, errors.ErrEmptyEnityName
	}

	// Checking that passed config is OUR.
	if _, ok := cfg.(*Config); !ok {
		return nil, db.ErrNotConfigPointer(Config{})
	}

	conn := &Enity{
		name:               name,
		ctx:                logger.Registrate(ctx),
		log:                logger.Get(ctx).GetLogger(db.ProvidersName, nil).With().Str("connection", name).Logger(),
		cfg:                cfg.(*Config).SetDefault(),
		connWatcherStopped: true,
	}

	conn.log.Info().Msgf("initializing connection...")

	return conn, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (c *Enity) Shutdown() error {
	c.log.Info().Msg("shutting down database connection")
	c.weAreShuttingDown = true

	if c.cfg.StartWatcher {
		for {
			if c.connWatcherStopped {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	} else {
		c.shutdown()
	}

	c.log.Info().Msg("connection shutted down")

	return nil
}

// Start starts connection workers and connection procedure itself.
func (c *Enity) Start() error {
	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if c.connWatcherStopped {
		if c.cfg.StartWatcher {
			c.connWatcherStopped = false
			go c.startWatcher()
		} else {
			// Manually start the connection once to establish connection
			_ = c.watcher()
		}
	}

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established and database migrations will be applied
// (or rolled back).
func (c *Enity) WaitForEstablishing() {
	for {
		if c.Conn != nil {
			break
		}

		c.log.Debug().Msg("connection isn't ready")
		time.Sleep(time.Millisecond * 100)
	}
}

// GetMetrics return map of the metrics from database connection
func (c *Enity) GetMetrics(prefix string) stats.MapMetricsOptions {
	_ = c.Service.GetMetrics(prefix)
	c.Metrics[prefix+"_"+c.name+"_status"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_status",
				Help: prefix + " " + c.name + " status link to " + utils.RedactedDSN(c.cfg.DSN),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(0)
			if c.Conn != nil {
				err := c.ping()
				if err == nil {
					(m.(prometheus.Gauge)).Set(1)
				}
			}
		},
	}

	return c.Metrics
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (c *Enity) GetReadyHandlers(prefix string) stats.MapCheckFunc {
	_ = c.Service.GetReadyHandlers(prefix)
	c.ReadyHandlers[strings.ToUpper(prefix+"_"+c.name+"_notfailed")] = func() (bool, string) {
		if c.Conn == nil {
			return false, "not connected"
		}

		if err := c.ping(); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return c.ReadyHandlers
}

func (c *Enity) newBucket() (*gridfs.Bucket, error) {
	if c.bucket != nil {
		return c.bucket, nil
	}

	bucket, err := gridfs.NewBucket(c.Conn.Database(c.dbName))
	if err != nil {
		return nil, err
	}

	c.bucket = bucket

	return c.bucket, nil
}

func writeToGridFile(fileName string, file multipart.File, gridFile *gridfs.UploadStream) (int, error) {
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
			return 0, ErrWriteFileToDatabase(fileName)
		}
		fileSize += size
	}
	gridFile.Close()
	return fileSize, nil
}

// ObjectIDFileName is a map between objectID and file name
type ObjectIDFileName map[string]string

func (c *Enity) WriteMultipart(fileprefix string, multipartForm *multipart.Form) (ObjectIDFileName, error) {
	result := make(ObjectIDFileName)
	for _, fileHeaders := range multipartForm.File {
		for _, fileHeader := range fileHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				return nil, err
			}

			bucket, err := c.newBucket()
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

			fileSize, err := writeToGridFile(fileHeader.Filename, file, gridFile)
			if err != nil {
				return nil, err
			}

			c.log.Debug().Msgf("write file to DB was successful; file size: %d \n", fileSize)

			result[gridFile.FileID.(primitive.ObjectID).Hex()] = filename
		}
	}

	return result, nil
}

func (c *Enity) GetFile(fileID string) (*bytes.Buffer, int64, error) {
	bucket, err := c.newBucket()
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

	c.log.Debug().Msgf("file size to download: %v\n", dStream)
	return &buf, dStream, nil
}

func (c *Enity) GetFileByName(fileName, fileprefix string) (*bytes.Buffer, int64, error) {
	bucket, err := c.newBucket()
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

	c.log.Debug().Msgf("file size to download: %v\n", dStream)
	return &buf, dStream, nil
}

func (c *Enity) Delete(fileID string) error {
	bucket, err := c.newBucket()
	if err != nil {
		return err
	}

	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return err
	}

	return bucket.Delete(objectID)
}
