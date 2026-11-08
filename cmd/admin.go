package cmd

import (
	"fmt"
	"net/rpc"
	"text/tabwriter"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/admin"
	"github.com/spf13/cobra"
)

var (
	nameFlag   string
	expiryFlag string
)

func init() {
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Manage admin users",
		Run:   printUsage,
	}

	adminListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all admins",
		RunE:  handleListAdmins,
	}

	adminCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new admin",
		RunE:  handleCreateAdmin,
	}
	adminCreateCmd.Flags().StringVarP(&nameFlag, "name", "n", "", "Name of the admin")
	adminCreateCmd.Flags().StringVarP(&expiryFlag, "expiry", "e", "", "Expiration date (yyyy-mm-dd) (at 00:00 UTC) of the admin")

	adminDeleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an admin",
		RunE:  handleDeleteAdmin,
	}
	adminDeleteCmd.Flags().StringVarP(&nameFlag, "name", "n", "", "Name of the admin")

	adminReactivateCmd := &cobra.Command{
		Use:   "reactivate",
		Short: "Reactivate an admin",
		RunE:  handleReactivateAdmin,
	}
	adminReactivateCmd.Flags().StringVarP(&nameFlag, "name", "n", "", "Name of the admin")

	adminExpiryCmd := &cobra.Command{
		Use:   "expiry",
		Short: "Set expiry date of an admin",
		RunE:  handleSetExpiryAdmin,
	}
	adminExpiryCmd.Flags().StringVarP(&expiryFlag, "expiry", "e", "", "Expiration date (yyyy-mm-dd) (at 00:00 UTC) of the admin")

	adminCmd.AddCommand(
		adminListCmd,
		adminCreateCmd,
		adminDeleteCmd,
		adminReactivateCmd,
		adminExpiryCmd,
	)

	rootCmd.AddCommand(adminCmd)
}

func getRPCClient() (*rpc.Client, error) {
	return rpc.DialHTTP("tcp", "127.0.0.1:8081")
}

func handleListAdmins(cmd *cobra.Command, args []string) error {
	var admins []admin.Admin

	client, err := getRPCClient()
	if err != nil {
		return err
	}

	err = client.Call("AdminHandler.List", 0, &admins)
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)

	w.Init(cmd.OutOrStdout(), 4, 4, 4, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, "ID\tName\tActivated at\tExpires at\n")

	timeFormat := "2006-01-02 15:04:05 MST"
	for _, admin := range admins {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t\n", admin.ID, admin.Name, admin.ActivatedAt.Format(timeFormat), admin.Expiry.Format(timeFormat))
	}

	return nil
}

func handleCreateAdmin(cmd *cobra.Command, args []string) error {
	var expiry time.Time
	if expiryFlag != "" {
		var err error
		expiry, err = time.Parse("2006-01-02", expiryFlag)
		if err != nil {
			return err
		}
	}

	admin := admin.Admin{
		Name:   nameFlag,
		Expiry: expiry,
	}

	client, err := getRPCClient()
	if err != nil {
		return err
	}

	var authToken string
	err = client.Call("AdminHandler.Create", admin, &authToken)
	if err != nil {
		return err
	}

	fmt.Printf("Admin \"%s\" created. Authorization token: %s\n", admin.Name, authToken)
	return nil
}

func handleDeleteAdmin(cmd *cobra.Command, args []string) error {
	admin := admin.Admin{
		Name: nameFlag,
	}

	client, err := getRPCClient()
	if err != nil {
		return err
	}

	err = client.Call("AdminHandler.Delete", admin, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Admin \"%s\" deleted.\n", admin.Name)
	return nil
}

func handleReactivateAdmin(cmd *cobra.Command, args []string) error {
	admin := admin.Admin{
		Name: nameFlag,
	}

	client, err := getRPCClient()
	if err != nil {
		return err
	}

	err = client.Call("AdminHandler.Reactivate", admin, &admin)
	if err != nil {
		return err
	}

	fmt.Printf("Admin \"%s\" reactivated. All tokens before %s are now invalidated. New authorization token: %s\n", admin.Name, admin.ActivatedAt.String(), admin.AuthorizationToken)
	return nil
}

func handleSetExpiryAdmin(cmd *cobra.Command, args []string) error {
	var expiry time.Time
	if expiryFlag != "" {
		var err error
		expiry, err = time.Parse("2006-01-02", expiryFlag)
		if err != nil {
			return err
		}
	}

	admin := admin.Admin{
		Name:   nameFlag,
		Expiry: expiry,
	}

	client, err := getRPCClient()
	if err != nil {
		return err
	}

	err = client.Call("AdminHandler.SetExpiry", admin, &admin)
	if err != nil {
		return err
	}

	fmt.Printf("Admin \"%s\" expiry set to %s.\n", admin.Name, admin.Expiry.String())
	return nil
}
