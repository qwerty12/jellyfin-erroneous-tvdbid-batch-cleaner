package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	jf "github.com/sj14/jellyfin-go/api"
)

func makeConfig() *jf.Configuration {
	server := os.Getenv("JELLYCTL_URL")
	if strings.TrimSpace(server) == "" {
		server = "http://127.0.0.1:8096"
	}

	token := os.Getenv("JELLYCTL_TOKEN")
	if strings.TrimSpace(token) == "" {
		log.Fatalln("JELLYCTL_TOKEN environment variable not set")
	}

	return &jf.Configuration{
		Servers:       jf.ServerConfigurations{{URL: server}},
		DefaultHeader: map[string]string{"Authorization": fmt.Sprintf(`MediaBrowser Token="%s"`, token)},
		Debug:         len(os.Args) == 2 && os.Args[1] == "--debug",
	}
}

func getFirstAdminUserId(ctx context.Context, client *jf.APIClient) string {
	users, _, err := client.UserAPI.GetUsers(ctx).IsDisabled(false).Execute()
	if err != nil {
		log.Fatalln("Error when calling `GetUsers`:", err)
	}

	for _, user := range users {
		policy := user.GetPolicy()
		if policy.GetIsAdministrator() {
			return user.GetId()
		}
	}

	return ""
}

func removeBadTvdbIdAndLock(ctx context.Context, client *jf.APIClient, itemId string, userId string) {
	item, _, err := client.UserLibraryAPI.GetItem(ctx, itemId).UserId(userId).Execute()
	if err != nil {
		log.Println("Error when calling `GetItem`:", err)
		return
	}

	item.ProviderIds["Tvdb"] = ""
	item.SetLockData(true)

	_, err = client.ItemUpdateAPI.UpdateItem(ctx, itemId).BaseItemDto(*item).Execute()
	if err != nil {
		log.Println("Error when calling `UpdateItem`:", err)
		return
	}
}

func main() {
	ctx := context.Background()
	client := jf.NewAPIClient(makeConfig())

	adminUserId := getFirstAdminUserId(ctx, client)
	if adminUserId == "" {
		log.Fatalln("No administrator found")
	}

	allItems, _, err := client.ItemsAPI.GetItems(ctx).
		Recursive(true).
		IncludeItemTypes([]jf.BaseItemKind{jf.BASEITEMKIND_SERIES}).
		Fields([]jf.ItemFields{jf.ITEMFIELDS_PROVIDER_IDS}).
		HasTvdbId(true).
		EnableTotalRecordCount(false).
		EnableImages(false).
		Execute()
	if err != nil {
		log.Fatalln("Error when calling `GetItems`:", err)
	}

	for _, item := range allItems.Items {
		if strings.HasPrefix(item.ProviderIds["Tvdb"], "tt") {
			fmt.Println(item.GetName(), item.ProviderIds["Tvdb"])
			removeBadTvdbIdAndLock(ctx, client, item.GetId(), adminUserId)
		}
	}
}
