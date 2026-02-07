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

func Collect(ctx context.Context) {
	serviceName, ok := ctx.Value(flag.LokiServiceNameKey{}).(string)
	if !ok {
		serviceName = "vmomi-event-source"
	}

	for {
		ch := make(chan *[]vmomi.Event)

		go func() {
			Watch(ctx, ch)
		}()

		Notify(ctx, ch, serviceName)

		// Retry after 3 seconds
		time.Sleep(time.Duration(3) * time.Second)
	}
}

func Watch(ctx context.Context, ch chan<- *[]vmomi.Event) {
	err := vmomi.Poll(ctx, nil, ch)
	if err != nil {
		slog.WarnContext(ctx, "Failed to poll events", "error", err)
	}
}

func Notify(ctx context.Context, ch <-chan *[]vmomi.Event, serviceName string) {
	for events := range ch {
		message := ToMessage(events, serviceName)

		err := Post(ctx, message)
		if err != nil {
			slog.WarnContext(ctx, "Failed to post event to Loki", "error", err)
		}
	}
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
	metadata := make([]*Metadata, 0, 9)
	//revive:enable:add-constant

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
