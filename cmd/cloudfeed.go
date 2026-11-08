package cmd

import (
	"errors"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/handlers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/spf13/cobra"
)

var (
	accountIDFlag   uint
	cloudFeedIDFlag uint
	startPeriodFlag string
	endPeriodFlag   string
)

func init() {
	cloudfeedCmd := &cobra.Command{
		Use:   "cloudfeed",
		Short: "Manage cloud feeds",
		Run:   printUsage,
	}

	cloudfeedDownloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Download data from a cloud feed",
		RunE:  handleCloudFeedDownload,
	}
	cloudfeedDownloadCmd.Flags().UintVarP(&accountIDFlag, "account-id", "a", 0, "Account ID")
	cloudfeedDownloadCmd.Flags().UintVarP(&cloudFeedIDFlag, "cloud-feed-id", "c", 0, "Cloud feed ID")
	cloudfeedDownloadCmd.Flags().StringVarP(&startPeriodFlag, "start", "s", "", "Start period (yyyy-mm-dd)")
	cloudfeedDownloadCmd.Flags().StringVarP(&endPeriodFlag, "end", "e", "", "End period (yyyy-mm-dd)")

	cloudfeedCmd.AddCommand(cloudfeedDownloadCmd)

	rootCmd.AddCommand(cloudfeedCmd)
}

func handleCloudFeedDownload(cmd *cobra.Command, args []string) error {
	if accountIDFlag == 0 {
		return errors.New("account ID is required")
	}
	if cloudFeedIDFlag == 0 {
		return errors.New("cloud feed ID is required")
	}
	if startPeriodFlag == "" {
		return errors.New("start period is required")
	}
	if endPeriodFlag == "" {
		endPeriodFlag = time.Now().Format("2006-01-02")
	}

	// Parse startPeriodFlag and endPeriodFlag as time strings
	startPeriod, err := time.Parse("2006-01-02", startPeriodFlag)
	if err != nil {
		return err
	}
	endPeriod, err := time.Parse("2006-01-02", endPeriodFlag)
	if err != nil {
		return err
	}

	downloadArgs := handlers.DownloadArgs{
		AccountID:   accountIDFlag,
		CloudFeedID: cloudFeedIDFlag,
		StartPeriod: needforheat.Time(startPeriod),
		EndPeriod:   needforheat.Time(endPeriod),
	}

	client, err := getRPCClient()
	if err != nil {
		return err
	}

	var reply string
	err = client.Call("CloudFeedHandler.Download", downloadArgs, &reply)
	if err != nil {
		return err
	}

	cmd.Println(reply)

	return nil
}
