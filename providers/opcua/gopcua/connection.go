package gopcua

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/utils"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	// Metrics
	stats.Service
	// OPC UA connection
	Conn *opcua.Client

	ctx  context.Context
	log  zerolog.Logger
	name string
	cfg  *Config

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

	conn.log.Info().Msgf("initializing enity " + name + "...")

	return conn, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (c *Enity) Shutdown() error {
	c.log.Info().Msg("shutting down opc ua connection watcher and queue worker")
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

type SubscribeResult struct {
	NotifyCh        chan *opcua.PublishNotificationData
	EventFieldNames []string
	Sub             *opcua.Subscription
}

func (c *Enity) SubscribeEvent(nodeID string) (result *SubscribeResult, err error) {
	result = &SubscribeResult{}
	result.NotifyCh = make(chan *opcua.PublishNotificationData)

	result.Sub, err = c.Conn.Subscribe(&opcua.SubscriptionParameters{
		Interval: c.cfg.Interval,
	}, result.NotifyCh)
	if err != nil {
		return nil, err
	}
	log.Printf("created subscription with id %v", result.Sub.SubscriptionID)

	uaid, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return nil, err
	}

	var miCreateRequest *ua.MonitoredItemCreateRequest
	miCreateRequest, result.EventFieldNames = eventRequest(uaid, c.cfg.Handle)
	res, err := result.Sub.Monitor(ua.TimestampsToReturnBoth, miCreateRequest)
	if err != nil || res.Results[0].StatusCode != ua.StatusOK {
		return nil, err
	}

	return result, nil
}

func (c *Enity) SubscribeValues(nodeID string) (result *SubscribeResult, err error) {
	result = &SubscribeResult{}
	result.NotifyCh = make(chan *opcua.PublishNotificationData)

	result.Sub, err = c.Conn.Subscribe(&opcua.SubscriptionParameters{
		Interval: c.cfg.Interval,
	}, result.NotifyCh)
	if err != nil {
		return nil, err
	}
	log.Printf("created subscription with id %v", result.Sub.SubscriptionID)

	uaid, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return nil, err
	}

	miCreateRequest := valueRequest(uaid, c.cfg.Handle)
	res, err := result.Sub.Monitor(ua.TimestampsToReturnBoth, miCreateRequest)
	if err != nil || res.Results[0].StatusCode != ua.StatusOK {
		return nil, err
	}

	return result, nil
}

func valueRequest(nodeID *ua.NodeID, handle uint32) *ua.MonitoredItemCreateRequest {
	return opcua.NewMonitoredItemCreateRequestWithDefaults(nodeID, ua.AttributeIDValue, handle)
}

func eventRequest(nodeID *ua.NodeID, handle uint32) (*ua.MonitoredItemCreateRequest, []string) {
	fieldNames := []string{"EventId", "EventType", "Severity", "Time", "Message"}
	selects := make([]*ua.SimpleAttributeOperand, len(fieldNames))

	for i, name := range fieldNames {
		selects[i] = &ua.SimpleAttributeOperand{
			TypeDefinitionID: ua.NewNumericNodeID(0, id.BaseEventType),
			BrowsePath:       []*ua.QualifiedName{{NamespaceIndex: 0, Name: name}},
			AttributeID:      ua.AttributeIDValue,
		}
	}

	wheres := &ua.ContentFilter{
		Elements: []*ua.ContentFilterElement{
			{
				FilterOperator: ua.FilterOperatorGreaterThanOrEqual,
				FilterOperands: []*ua.ExtensionObject{
					{
						EncodingMask: 1,
						TypeID: &ua.ExpandedNodeID{
							NodeID: ua.NewNumericNodeID(0, id.SimpleAttributeOperand_Encoding_DefaultBinary),
						},
						Value: ua.SimpleAttributeOperand{
							TypeDefinitionID: ua.NewNumericNodeID(0, id.BaseEventType),
							BrowsePath:       []*ua.QualifiedName{{NamespaceIndex: 0, Name: "Severity"}},
							AttributeID:      ua.AttributeIDValue,
						},
					},
					{
						EncodingMask: 1,
						TypeID: &ua.ExpandedNodeID{
							NodeID: ua.NewNumericNodeID(0, id.LiteralOperand_Encoding_DefaultBinary),
						},
						Value: ua.LiteralOperand{
							Value: ua.MustVariant(uint16(0)),
						},
					},
				},
			},
		},
	}

	filter := ua.EventFilter{
		SelectClauses: selects,
		WhereClause:   wheres,
	}

	filterExtObj := ua.ExtensionObject{
		EncodingMask: ua.ExtensionObjectBinary,
		TypeID: &ua.ExpandedNodeID{
			NodeID: ua.NewNumericNodeID(0, id.EventFilter_Encoding_DefaultBinary),
		},
		Value: filter,
	}

	req := &ua.MonitoredItemCreateRequest{
		ItemToMonitor: &ua.ReadValueID{
			NodeID:       nodeID,
			AttributeID:  ua.AttributeIDEventNotifier,
			DataEncoding: &ua.QualifiedName{},
		},
		MonitoringMode: ua.MonitoringModeReporting,
		RequestedParameters: &ua.MonitoringParameters{
			ClientHandle:     handle,
			DiscardOldest:    true,
			Filter:           &filterExtObj,
			QueueSize:        10,
			SamplingInterval: 1.0,
		},
	}

	return req, fieldNames
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
