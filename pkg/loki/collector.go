package loki

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/9506hqwy/vmomi-event-source/pkg/flag"
	"github.com/9506hqwy/vmomi-event-source/pkg/vmomi"
)

const Empty = int(0)

func Collect(ctx context.Context) {
	serviceName, ok := ctx.Value(flag.LokiServiceNameKey{}).(string)
	if !ok {
		serviceName = "vmomi-event-source"
	}

	latestKey := int32(Empty)

	for {
		ch := make(chan *[]vmomi.Event)

		go func() {
			Watch(ctx, ch, latestKey)
		}()

		latestKey = Notify(ctx, ch, serviceName, latestKey)

		// Retry after 3 seconds
		time.Sleep(time.Duration(3) * time.Second)
	}
}

func Watch(ctx context.Context, ch chan<- *[]vmomi.Event, previousKey int32) {
	err := vmomi.Poll(ctx, nil, ch, previousKey)
	if err != nil {
		slog.WarnContext(ctx, "Failed to poll events", "error", err)
	}
}

func Notify(
	ctx context.Context,
	ch <-chan *[]vmomi.Event,
	serviceName string,
	previousKey int32,
) int32 {
	latestKey := previousKey

	for events := range ch {
		latestKey = getLastEventKey(events)

		message := ToMessage(events, serviceName)

		err := Post(ctx, message)
		if err != nil {
			slog.WarnContext(ctx, "Failed to post event to Loki", "error", err)
		}
	}

	return latestKey
}

func ToMessage(events *[]vmomi.Event, serviceName string) *Message {
	return &Message{
		Streams: ToStreams(events, serviceName),
	}
}

func ToStreams(events *[]vmomi.Event, serviceName string) []*Stream {
	streams := make([]*Stream, len(*events))

	for i, event := range *events {
		streams[i] = ToStream(&event, serviceName)
	}

	return streams
}

func ToStream(event *vmomi.Event, serviceName string) *Stream {
	metadata := CreateMetadata(event)

	return &Stream{
		Labels: fmt.Sprintf(`{service_name="%s", severity="%s"}`, serviceName, event.Severity),
		Entries: []*Entry{
			{
				Timestamp:          timestamppb.New(event.CreatedTime),
				Line:               event.FullFormattedMessage,
				StructuredMetadata: metadata,
			},
		},
	}
}

//revive:disable:cognitive-complexity

func CreateMetadata(event *vmomi.Event) []*Metadata {
	//revive:disable:add-constant
	metadata := make([]*Metadata, 0, 10)
	//revive:enable:add-constant

	metadata = append(metadata, &Metadata{
		Name:  "internal_key",
		Value: fmt.Sprint(event.Key),
	})

	if event.ComputeResource != nil && *event.ComputeResource != *event.Host {
		metadata = append(metadata, &Metadata{
			Name:  "cluster",
			Value: *event.ComputeResource,
		})
	}

	if event.Datacenter != nil {
		metadata = append(metadata, &Metadata{
			Name:  "datacenter",
			Value: *event.Datacenter,
		})
	}

	if event.Datastore != nil {
		metadata = append(metadata, &Metadata{
			Name:  "datastore",
			Value: *event.Datastore,
		})
	}

	if event.DistributedVirtualSwitch != nil {
		metadata = append(metadata, &Metadata{
			Name:  "distributed_virtual_switch",
			Value: *event.DistributedVirtualSwitch,
		})
	}

	if event.Host != nil {
		metadata = append(metadata, &Metadata{
			Name:  "host",
			Value: *event.Host,
		})
	}

	if event.Network != nil {
		metadata = append(metadata, &Metadata{
			Name:  "network",
			Value: *event.Network,
		})
	}

	metadata = append(metadata, &Metadata{
		Name:  "user",
		Value: event.UserName,
	})

	if event.VM != nil {
		metadata = append(metadata, &Metadata{
			Name:  "vm",
			Value: *event.VM,
		})
	}

	metadata = append(metadata, &Metadata{
		Name:  "event_type_id",
		Value: event.EventTypeID,
	})

	return metadata
}

//revive:enable:cognitive-complexity

func getLastEventKey(events *[]vmomi.Event) int32 {
	//revive:disable:add-constant
	lastEvent := (*events)[len(*events)-1]
	//revive:enable:add-constant
	return lastEvent.Key
}
