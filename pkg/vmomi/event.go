package vmomi

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	sx "github.com/9506hqwy/vmomi-event-source/pkg/vmomi/sessionex"
)

const (
	MaxEventCount    = int32(1000)
	MaxObjectUpdates = int32(1)
)

const Empty = int(0)

type Event struct {
	Key                      int32
	ComputeResource          *string
	CreatedTime              time.Time
	Datacenter               *string
	Datastore                *string
	DistributedVirtualSwitch *string
	FullFormattedMessage     string
	Host                     *string
	Network                  *string
	UserName                 string
	VM                       *string
	Severity                 string
	EventTypeID              string
}

type EventInfo struct {
	Key             string
	Description     string
	Category        string
	LongDescription EventLongDescription
}

type EnumeratedType struct {
	Key    string
	Labels []string
}

type EventLongDescription struct {
	Description string       `xml:"description"`
	Causes      []EventCause `xml:"cause"`
}

type EventCause struct {
	Description string   `xml:"description"`
	Actions     []string `xml:"action"`
}

//revive:disable:cognitive-complexity

func (e Event) Target() string {
	path := []string{}

	if e.Datacenter != nil {
		path = append(path, *e.Datacenter)
	}

	if e.DistributedVirtualSwitch != nil {
		path = append(path, *e.DistributedVirtualSwitch)
	}

	if e.ComputeResource != nil {
		path = append(path, *e.ComputeResource)
	}

	if e.Host != nil && *e.ComputeResource != *e.Host {
		path = append(path, *e.Host)
	}

	if e.Network != nil {
		path = append(path, *e.Network)
	}

	if e.Datastore != nil {
		path = append(path, *e.Datastore)
	}

	if e.VM != nil {
		path = append(path, *e.VM)
	}

	return strings.Join(path, "/")
}

//revive:enable:cognitive-complexity

func Query(ctx context.Context) ([]Event, error) {
	c, err := login(ctx)
	if err != nil {
		return nil, err
	}

	defer sx.Logout(ctx, c)

	em := event.NewManager(c)

	collector, err := createEventCollector(ctx, em)
	if err != nil {
		return nil, err
	}

	defer destroyEventCollector(ctx, collector)

	events, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) ([]types.BaseEvent, error) {
			return collector.LatestPage(cctx)
		},
	)
	if err != nil {
		return nil, err
	}

	e, err := getEventManager(ctx, c)
	if err != nil {
		return nil, err
	}

	return ToEvents(e, &events), nil
}

//revive:disable:cognitive-complexity

func Poll(
	ctx context.Context,
	maxWaitSeconds *int32,
	ch chan<- *[]Event,
	previousKey int32,
) error {
	c, err := login(ctx)
	if err != nil {
		close(ch)
		return err
	}

	defer sx.Logout(ctx, c)

	em := event.NewManager(c)

	collector, err := createEventCollector(ctx, em)
	if err != nil {
		close(ch)
		return err
	}

	defer destroyEventCollector(ctx, collector)

	events, err := readyEventCollector(ctx, c, collector)
	if err != nil {
		close(ch)
		return err
	}

	err = sendAfterKey(previousKey, events, ch)
	if err != nil {
		close(ch)
		return err
	}

	waiter, filter, err := createLatestEventWatcher(ctx, em)
	if err != nil {
		close(ch)
		return err
	}

	defer destroyPropertyCollector(ctx, waiter)
	defer destroyPropertyFilter(ctx, filter)

	err = waitUpdateForLatestEvent(
		ctx,
		c,
		waiter,
		maxWaitSeconds,
		collector,
		func(evts *[]Event) {
			if len(*evts) != Empty {
				ch <- evts
			}
		},
	)
	if err != nil {
		close(ch)
		return err
	}

	close(ch)
	return nil
}

//revive:enable:cognitive-complexity

func GetCategories(ctx context.Context) ([]string, error) {
	c, err := login(ctx)
	if err != nil {
		return nil, err
	}

	defer sx.Logout(ctx, c)

	e, err := getEventManager(ctx, c)
	if err != nil {
		return nil, err
	}

	labels := make([]string, len(e.Description.Category))
	for i, category := range e.Description.Category {
		labels[i] = category.GetElementDescription().Label
	}

	return labels, nil
}

func GetEnumeratedTypes(ctx context.Context) ([]EnumeratedType, error) {
	c, err := login(ctx)
	if err != nil {
		return nil, err
	}

	defer sx.Logout(ctx, c)

	e, err := getEventManager(ctx, c)
	if err != nil {
		return nil, err
	}

	enumerateds := make([]EnumeratedType, len(e.Description.EnumeratedTypes))
	for i, enumerated := range e.Description.EnumeratedTypes {
		labels := make([]string, len(enumerated.Tags))
		for j, tag := range enumerated.Tags {
			labels[j] = tag.GetElementDescription().Label
		}

		enumerateds[i] = EnumeratedType{
			Key:    enumerated.Key,
			Labels: labels,
		}
	}

	return enumerateds, nil
}

func GetEventInfo(ctx context.Context) ([]EventInfo, error) {
	c, err := login(ctx)
	if err != nil {
		return nil, err
	}

	defer sx.Logout(ctx, c)

	locale, err := sx.GetLocale(ctx, c)
	if err != nil {
		return nil, err
	}

	e, err := getEventManager(ctx, c)
	if err != nil {
		return nil, err
	}

	l, err := GetLocalizationManager(ctx, c)
	if err != nil {
		return nil, err
	}

	ex, err := GetExtensionManager(ctx, c)
	if err != nil {
		return nil, err
	}

	eventEx := ListExtentionEvent(ex)

	info := make([]EventInfo, len(e.Description.EventInfo)+len(eventEx))
	collectEvent(ctx, e, &info)

	infoEx := info[len(e.Description.EventInfo):]
	CollectExtensionEvent(ctx, c, l, *locale, eventEx, &infoEx)

	return info, nil
}

func ToEvents(em *mo.EventManager, events *[]types.BaseEvent) []Event {
	metrics := make([]Event, len(*events))
	for i, e := range *events {
		metrics[i] = ToEvent(em, e)
	}

	sort.Slice(
		metrics,
		func(i, j int) bool {
			return metrics[j].CreatedTime.After(metrics[i].CreatedTime)
		},
	)

	return metrics
}

func ToEvent(em *mo.EventManager, e types.BaseEvent) Event {
	evt := *e.GetEvent()
	model := Event{
		Key:                  evt.Key,
		CreatedTime:          evt.CreatedTime,
		FullFormattedMessage: evt.FullFormattedMessage,
		UserName:             evt.UserName,
		Severity:             getEventSeverity(em, &e),
		EventTypeID:          getEventTypeID(&e),
	}

	if evt.ComputeResource != nil {
		model.ComputeResource = &evt.ComputeResource.Name
	}

	if evt.Datacenter != nil {
		model.Datacenter = &evt.Datacenter.Name
	}

	if evt.Ds != nil {
		model.Datastore = &evt.Ds.Name
	}

	if evt.Dvs != nil {
		model.DistributedVirtualSwitch = &evt.Dvs.Name
	}

	if evt.Host != nil {
		model.Host = &evt.Host.Name
	}

	if evt.Net != nil {
		model.Network = &evt.Net.Name
	}

	if evt.Vm != nil {
		model.VM = &evt.Vm.Name
	}

	return model
}

func getEventManager(ctx context.Context, c *vim25.Client) (*mo.EventManager, error) {
	pc := property.DefaultCollector(c)

	var e mo.EventManager
	_, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) (int, error) {
			return 0, pc.RetrieveOne(
				cctx,
				*c.ServiceContent.EventManager,
				[]string{"description"},
				&e,
			)
		},
	)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func getEventSeverity(em *mo.EventManager, evt *types.BaseEvent) string {
	if e, ok := (*evt).(*types.EventEx); ok && e.Severity != "" {
		return e.Severity
	}

	typeID := getEventTypeID(evt)
	return getCategoryKey(em, typeID)
}

func getCategoryKey(em *mo.EventManager, typeID string) string {
	for _, info := range em.Description.EventInfo {
		if info.Key == typeID && containsCategoryKey(em, info.Category) {
			return info.Category
		}
	}

	// Default severity
	return "info"
}

func containsCategoryKey(em *mo.EventManager, key string) bool {
	for _, category := range em.Description.Category {
		if category.GetElementDescription().Key == key {
			return true
		}
	}

	return false
}

func getEventTypeID(evt *types.BaseEvent) string {
	if e, ok := (*evt).(*types.EventEx); ok {
		return e.EventTypeId
	}

	if e, ok := (*evt).(*types.ExtendedEvent); ok {
		return e.EventTypeId
	}

	typeID := strings.TrimPrefix(fmt.Sprintf("%T", *evt), "*types.")
	return typeID
}

func collectEvent(ctx context.Context, e *mo.EventManager, info *[]EventInfo) {
	for i, evInfo := range e.Description.EventInfo {
		longDesc := EventLongDescription{}
		if len(evInfo.LongDescription) != Empty {
			err := xml.Unmarshal([]byte(evInfo.LongDescription), &longDesc)
			if err != nil {
				slog.WarnContext(
					ctx,
					"Could not deserialize long description",
					"error", err,
					"description", evInfo.LongDescription,
				)
			}
		}

		(*info)[i] = EventInfo{
			Key:             evInfo.Key,
			Description:     evInfo.Description,
			Category:        evInfo.Category,
			LongDescription: longDesc,
		}
	}
}

func createEventCollector(
	ctx context.Context,
	em *event.Manager,
) (*event.HistoryCollector, error) {
	filter := types.EventFilterSpec{}

	collector, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) (*event.HistoryCollector, error) {
			return em.CreateCollectorForEvents(cctx, filter)
		},
	)
	if err != nil {
		return nil, err
	}

	return collector, nil
}

func createLatestEventWatcher(
	ctx context.Context,
	em *event.Manager,
) (*property.Collector, *property.Filter, error) {
	pm := property.DefaultCollector(em.Client())

	waiter, err := pm.Create(ctx)
	if err != nil {
		return nil, nil, err
	}

	spec := types.PropertyFilterSpec{
		ObjectSet: []types.ObjectSpec{
			{
				Obj: em.Reference(),
			},
		},
		PropSet: []types.PropertySpec{
			{
				Type:    "EventManager",
				PathSet: []string{"latestEvent"},
			},
		},
	}

	req := types.CreateFilter{
		Spec:           spec,
		PartialUpdates: false,
	}

	filter, err := waiter.CreateFilter(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return waiter, filter, nil
}

func readyEventCollector(
	ctx context.Context,
	c *vim25.Client,
	collector *event.HistoryCollector,
) (*[]Event, error) {
	_, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) (int, error) {
			return 0, collector.SetPageSize(cctx, MaxEventCount)
		},
	)
	if err != nil {
		return nil, err
	}

	events, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) ([]types.BaseEvent, error) {
			return collector.ReadNextEvents(cctx, MaxEventCount)
		},
	)
	if err != nil {
		return nil, err
	}

	e, err := getEventManager(ctx, c)
	if err != nil {
		return nil, err
	}

	es := ToEvents(e, &events)
	return &es, nil
}

func destroyEventCollector(ctx context.Context, collector *event.HistoryCollector) error {
	_, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) (int, error) {
			return 0, collector.Destroy(cctx)
		},
	)
	return err
}

func destroyPropertyCollector(ctx context.Context, collector *property.Collector) error {
	_, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) (int, error) {
			return 0, collector.Destroy(cctx)
		},
	)
	return err
}

func destroyPropertyFilter(ctx context.Context, filter *property.Filter) error {
	_, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) (int, error) {
			return 0, filter.Destroy(cctx)
		},
	)
	return err
}

func waitUpdateForLatestEvent(
	ctx context.Context,
	c *vim25.Client,
	waiter *property.Collector,
	maxWaitSeconds *int32,
	collector *event.HistoryCollector,
	onUpdatesFn func(*[]Event),
) error {
	opt := property.WaitOptions{
		Options: &types.WaitOptions{
			MaxObjectUpdates: MaxObjectUpdates,
			MaxWaitSeconds:   maxWaitSeconds,
		},
	}

	e, err := getEventManager(ctx, c)
	if err != nil {
		return err
	}

	err = waiter.WaitForUpdatesEx(ctx, &opt, func(_ []types.ObjectUpdate) bool {
		evts, err := sx.ExecCallAPI(
			ctx,
			func(cctx context.Context) ([]types.BaseEvent, error) {
				return collector.ReadNextEvents(cctx, MaxEventCount)
			},
		)
		if err != nil {
			slog.WarnContext(ctx, "Failed to read next events", "error", err)
			return false
		}

		es := ToEvents(e, &evts)
		onUpdatesFn(&es)

		return false
	})
	if err != nil {
		return err
	}

	return nil
}

func sendAfterKey(key int32, events *[]Event, ch chan<- *[]Event) error {
	if key == int32(Empty) {
		return nil
	}

	if len(*events) == Empty {
		return nil
	}

	ch <- filterAfterKey(key, events)

	return nil
}

func filterAfterKey(key int32, events *[]Event) *[]Event {
	found := false
	targets := make([]Event, Empty, len(*events))
	for _, e := range *events {
		if found {
			targets = append(targets, e)
		}

		if e.Key == key {
			found = true
		}
	}

	return &targets
}
