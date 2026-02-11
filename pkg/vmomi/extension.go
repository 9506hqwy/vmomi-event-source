package vmomi

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"

	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"

	sx "github.com/9506hqwy/vmomi-event-source/pkg/vmomi/sessionex"
)

type ExtensionEvent struct {
	ModuleName      string
	EventID         string
	EventTypeSchema string
}

type extensionEventType struct {
	EventTypeID string                      `xml:"eventTypeID"`
	Description string                      `xml:"description"`
	Arguments   extensionEventTypeArguments `xml:"arguments,omitempty"`
}

type extensionEventTypeArguments struct {
	Argument []extensionEventTypeArgument `xml:"argument"`
}

type extensionEventTypeArgument struct {
	Name string `xml:"name"`
	Type string `xml:"type"`
}

func GetExtensionManager(ctx context.Context, c *vim25.Client) (*mo.ExtensionManager, error) {
	if c.ServiceContent.ExtensionManager == nil {
		return nil, nil
	}

	pc := property.DefaultCollector(c)

	var e mo.ExtensionManager
	_, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) (int, error) {
			return 0, pc.RetrieveOne(
				cctx,
				*c.ServiceContent.ExtensionManager,
				[]string{"extensionList"},
				&e,
			)
		},
	)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func ListExtentionEvent(em *mo.ExtensionManager) []*ExtensionEvent {
	eventEx := []*ExtensionEvent{}
	if em == nil {
		return eventEx
	}

	for _, ext := range em.ExtensionList {
		if ext.EventList != nil {
			for _, evInfo := range ext.EventList {
				e := ExtensionEvent{
					ModuleName:      ext.Key,
					EventID:         evInfo.EventID,
					EventTypeSchema: evInfo.EventTypeSchema,
				}
				eventEx = append(eventEx, &e)
			}
		}
	}

	return eventEx
}

func CollectExtensionEvent(
	ctx context.Context,
	c *vim25.Client,
	lm *mo.LocalizationManager,
	locale string,
	evts []*ExtensionEvent,
	info *[]EventInfo,
) {
	for i, evInfo := range evts {
		eventType := extensionEventType{}
		if len(evInfo.EventTypeSchema) != Empty {
			err := xml.Unmarshal([]byte(evInfo.EventTypeSchema), &eventType)
			if err != nil {
				slog.WarnContext(
					ctx,
					"Could not deserialize event type schema",
					"error", err,
					"schema", evInfo.EventTypeSchema,
				)
				eventType = extensionEventType{}
			}
		}

		(*info)[i] = EventInfo{
			Key:             evInfo.EventID,
			Description:     eventType.Description,
			Category:        getExtensionEventSeverity(ctx, c, lm, locale, evInfo),
			LongDescription: EventLongDescription{},
		}
	}
}

func getExtensionEventSeverity(
	ctx context.Context,
	c *vim25.Client,
	lm *mo.LocalizationManager,
	locale string,
	event *ExtensionEvent,
) string {
	// Default severity
	catalog := "info"

	if lm == nil {
		return catalog
	}

	catalogKey := fmt.Sprintf("%s.category", event.EventID)
	catalogValue, err := GetLocalizationCatalogValue(
		ctx,
		c,
		lm,
		locale,
		event.ModuleName,
		catalogKey,
	)
	if err != nil {
		slog.WarnContext(
			ctx,
			"Could not get localization",
			"error", err,
			"locale", locale,
			"key", catalogKey,
		)
	}

	if catalogValue != nil {
		catalog = *catalogValue
	}

	return catalog
}
