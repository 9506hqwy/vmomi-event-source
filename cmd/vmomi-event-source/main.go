package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/9506hqwy/vmomi-event-source/pkg/flag"
	"github.com/9506hqwy/vmomi-event-source/pkg/loki"
	"github.com/9506hqwy/vmomi-event-source/pkg/vmomi"
)

var version = "<version>"
var commit = "<commit>"

//revive:disable:deep-exit

var rootCmd = &cobra.Command{
	Use:     "vmomi-event-source",
	Short:   "VMOMI Event Source",
	Long:    "VMOMI Event Source",
	Version: fmt.Sprintf("%s\nCommit: %s", version, commit),
}

var categoryCmd = &cobra.Command{
	Use:     "category",
	Short:   "VMOMI Event Source Category",
	Long:    "VMOMI Event Source Category",
	Version: fmt.Sprintf("%s\nCommit: %s", version, commit),
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()
		ctx = fromArgument(ctx)

		categories, err := vmomi.GetCategories(ctx)
		if err != nil {
			log.Fatalf("GetCategories error: %v", err)
		}

		for _, category := range categories {
			_, err := fmt.Println(category)
			if err != nil {
				log.Fatalf("Print error: %v", err)
			}
		}
	},
}

var enumeratedCmd = &cobra.Command{
	Use:     "enumerated",
	Short:   "VMOMI Event Source Enumerated",
	Long:    "VMOMI Event Source Enumerated",
	Version: fmt.Sprintf("%s\nCommit: %s", version, commit),
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()
		ctx = fromArgument(ctx)

		enumeratedTypes, err := vmomi.GetEnumeratedTypes(ctx)
		if err != nil {
			log.Fatalf("GetEnumeratedTypes error: %v", err)
		}

		for _, enumerated := range enumeratedTypes {
			_, err := fmt.Println(
				enumerated.Key,
				"(",
				strings.Join(enumerated.Labels, ", "),
				")",
			)

			if err != nil {
				log.Fatalf("Print error: %v", err)
			}
		}
	},
}

var infoCmd = &cobra.Command{
	Use:     "info",
	Short:   "VMOMI Event Source Info",
	Long:    "VMOMI Event Source Info",
	Version: fmt.Sprintf("%s\nCommit: %s", version, commit),
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()
		ctx = fromArgument(ctx)

		info, err := vmomi.GetEventInfo(ctx)
		if err != nil {
			log.Fatalf("GetEventInfo error: %v", err)
		}

		for _, i := range info {
			_, err := fmt.Println(
				i.Key,
				i.Category,
				i.Description,
			)
			if err != nil {
				log.Fatalf("Print error: %v", err)
			}

			if i.LongDescription.Description != "" {
				_, err := fmt.Println(
					"  Long Description:",
					i.LongDescription.Description,
				)
				if err != nil {
					log.Fatalf("Print error: %v", err)
				}

				for _, c := range i.LongDescription.Causes {
					_, err := fmt.Println("  Cause:", c.Description)
					if err != nil {
						log.Fatalf("Print error: %v", err)
					}
				}
			}
		}
	},
}

var eventCmd = &cobra.Command{
	Use:     "event",
	Short:   "VMOMI Event Source Event",
	Long:    "VMOMI Event Source Event",
	Version: fmt.Sprintf("%s\nCommit: %s", version, commit),
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()
		ctx = fromArgument(ctx)

		events, err := vmomi.Query(ctx)
		if err != nil {
			log.Fatalf("Query error: %v", err)
		}

		for _, event := range events {
			_, err = fmt.Printf(
				"%v\tuser=%v\tseverity=%v\ttarget=%v\tmessage=%v\n",
				event.CreatedTime,
				event.UserName,
				event.Severity,
				event.Target(),
				event.FullFormattedMessage)
			if err != nil {
				log.Fatalf("Print error: %v", err)
			}
		}
	},
}

var waitCmd = &cobra.Command{
	Use:     "wait",
	Short:   "VMOMI Event Source Wait",
	Long:    "VMOMI Event Source Wait",
	Version: fmt.Sprintf("%s\nCommit: %s", version, commit),
	Run: func(cmd *cobra.Command, _ []string) {
		timeout, err := cmd.Flags().GetInt32("timeout")
		if err != nil {
			log.Fatalf("Get timeout error: %v", err)
		}

		ctx := context.Background()
		ctx = fromArgument(ctx)

		ch := make(chan *[]vmomi.Event)

		go func() {
			err := vmomi.Poll(ctx, &timeout, ch, 0)
			if err != nil {
				log.Fatalf("Poll error: %v", err)
			}
		}()

		for events := range ch {
			for _, event := range *events {
				_, err := fmt.Printf(
					"%v\tuser=%v\tseverity=%v\ttarget=%v\tmessage=%v\n",
					event.CreatedTime,
					event.UserName,
					event.Severity,
					event.Target(),
					event.FullFormattedMessage)
				if err != nil {
					log.Fatalf("Print error: %v", err)
				}
			}
		}
	},
}

var lokiCmd = &cobra.Command{
	Use:     "loki",
	Short:   "VMOMI Event Source Loki",
	Long:    "VMOMI Event Source Loki",
	Version: fmt.Sprintf("%s\nCommit: %s", version, commit),
}

var lokiTestCmd = &cobra.Command{
	Use:     "test",
	Short:   "VMOMI Event Source Loki Test",
	Long:    "VMOMI Event Source Loki Test",
	Version: fmt.Sprintf("%s\nCommit: %s", version, commit),
	Run: func(cmd *cobra.Command, _ []string) {
		message, err := cmd.Flags().GetString("message")
		if err != nil {
			log.Fatalf("Get message error: %v", err)
		}

		ctx := context.Background()
		ctx = fromArgument(ctx)

		msg := loki.Message{
			Streams: []*loki.Stream{
				{
					Entries: []*loki.Entry{
						{
							Timestamp: timestamppb.Now(),
							Line:      message,
						},
					},
				},
			},
		}

		err = loki.Post(ctx, &msg)
		if err != nil {
			log.Fatalf("Post error: %v", err)
		}
	},
}

var lokiCollectCmd = &cobra.Command{
	Use:     "collect",
	Short:   "VMOMI Event Source Loki Collect",
	Long:    "VMOMI Event Source Loki Collect",
	Version: fmt.Sprintf("%s\nCommit: %s", version, commit),
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()
		ctx = fromArgument(ctx)
		loki.Collect(ctx)
	},
}

//revive:enable:deep-exit

//revive:disable:line-length-limit

func fromArgument(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, flag.TargetURLKey{}, viper.GetString("target_url"))
	ctx = context.WithValue(ctx, flag.TargetUserKey{}, viper.GetString("target_user"))
	ctx = context.WithValue(ctx, flag.TargetPasswordKey{}, viper.GetString("target_password"))
	ctx = context.WithValue(ctx, flag.TargetNoVerifySSLKey{}, viper.GetBool("target_no_verify_ssl"))
	ctx = context.WithValue(ctx, flag.TargetTimeoutKey{}, viper.GetInt("target_timeout"))
	ctx = context.WithValue(ctx, flag.LogLevelKey{}, viper.GetString("log_level"))

	ctx = context.WithValue(ctx, flag.LokiURLKey{}, viper.GetString("loki_url"))
	ctx = context.WithValue(ctx, flag.LokiTenantIDKey{}, viper.GetString("loki_tenant"))
	ctx = context.WithValue(ctx, flag.LokiNoVerifySSLKey{}, viper.GetBool("loki_no_verify_ssl"))
	ctx = context.WithValue(ctx, flag.LokiServiceNameKey{}, viper.GetString("loki_service_name"))
	return ctx
}

//revive:disable:add-constant

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("url", "https://127.0.0.1/sdk", "vSphere server URL.")
	rootCmd.PersistentFlags().String("user", "", "vSphere server username.")
	rootCmd.PersistentFlags().String("password", "", "vSphere server password.")
	rootCmd.PersistentFlags().Bool("no-verify-ssl", false, "Skip SSL verification.")
	rootCmd.PersistentFlags().Int("timeout", 10, "API call timeout seconds.")
	rootCmd.PersistentFlags().String("log-level", "INFO", "Log level.")

	waitCmd.Flags().Int32("timeout", 60, "Timeout in seconds.")

	lokiCmd.PersistentFlags().String("loki-url", "http://127.0.0.1:3100/loki/api/v1/push", "Loki URL.")
	lokiCmd.PersistentFlags().String("tenant", "", "Loki tenant.")
	lokiCmd.PersistentFlags().Bool("loki-no-verify-ssl", false, "Skip SSL verification.")
	lokiCmd.PersistentFlags().String("loki-service-name", "vmomi-event-source", "Loki service name.")

	lokiTestCmd.Flags().String("message", "Test message", "Message to send.")

	rootCmd.AddCommand(categoryCmd)
	rootCmd.AddCommand(enumeratedCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(eventCmd)
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(lokiCmd)

	lokiCmd.AddCommand(lokiTestCmd)
	lokiCmd.AddCommand(lokiCollectCmd)

	viper.BindPFlag("target_url", rootCmd.PersistentFlags().Lookup("url"))
	viper.BindPFlag("target_user", rootCmd.PersistentFlags().Lookup("user"))
	viper.BindPFlag("target_password", rootCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("target_no_verify_ssl", rootCmd.PersistentFlags().Lookup("no-verify-ssl"))
	viper.BindPFlag("target_timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("log_level", rootCmd.Flags().Lookup("log-level"))

	viper.BindPFlag("loki_url", lokiCmd.PersistentFlags().Lookup("loki-url"))
	viper.BindPFlag("loki_tenant", lokiCmd.PersistentFlags().Lookup("tenant"))
	viper.BindPFlag("loki_no_verify_ssl", lokiCmd.PersistentFlags().Lookup("loki-no-verify-ssl"))
	viper.BindPFlag("loki_service_name", lokiCmd.PersistentFlags().Lookup("loki-service-name"))
}

//revive:enable:add-constant

//revive:enable:line-length-limit

func initConfig() {
	viper.SetEnvPrefix("vmomi_event_source")
	viper.AutomaticEnv()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
