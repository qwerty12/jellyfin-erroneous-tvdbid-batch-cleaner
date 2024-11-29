package main

import (
	"context"
	"log"
	"os"
	"strings"

	jf "github.com/sj14/jellyfin-go/api"
)

func makeConfig() *jf.Configuration {
	server := os.Getenv("JELLYCTL_URL")
	if server == "" {
		server = "http://127.0.0.1:8096"
	}

	token := os.Getenv("JELLYCTL_TOKEN")
	if token == "" {
		log.Fatalln("JELLYCTL_TOKEN environment variable not set")
	}

	return &jf.Configuration{
		Servers:       jf.ServerConfigurations{{URL: server}},
		DefaultHeader: map[string]string{"Authorization": `MediaBrowser Token="` + token + `"`},
		Debug:         len(os.Args) == 2 && os.Args[1] == "--debug",
	}
}

func getFirstAdminUserId(ctx context.Context, client *jf.APIClient) string {
	users, _, err := client.UserAPI.GetUsers(ctx).IsDisabled(false).Execute()
	if err != nil {
		log.Fatalln("Error when calling `GetUsers`:", err)
	}

	for _, user := range users {
		if policy := user.GetPolicy(); policy.GetIsAdministrator() {
			return user.GetId()
		}
	}

	return ""
}

func removeBadTvdbIdAndLock(ctx context.Context, client *jf.APIClient, itemId, userId string) {
	item, _, err := client.UserLibraryAPI.GetItem(ctx, itemId).UserId(userId).Execute()
	if err != nil {
		log.Printf("Error when calling `GetItem` for item ID `%v`: %v\n", itemId, err)
		return
	}

	item.ProviderIds["Tvdb"] = ""
	item.SetLockData(true)

	_, err = client.ItemUpdateAPI.UpdateItem(ctx, itemId).BaseItemDto(*item).Execute()
	if err != nil {
		log.Printf("Error when calling `UpdateItem` for item ID `%v`: %v\n", itemId, err)
		return
	}

	if /*item.GetType() != jf.BASEITEMKIND_SERIES ||*/ item.GetChildCount() != 1 {
		return
	}

	seasons, _, err := client.TvShowsAPI.GetSeasons(ctx, itemId).UserId(userId).Execute()
	if err != nil {
		log.Printf("Unable to get seasons for `%v`: %v\n", itemId, err)
		return
	}

	for _, season := range seasons.Items {
		seasonId := season.GetId()
		seasonItem, _, err := client.UserLibraryAPI.GetItem(ctx, seasonId).UserId(userId).Execute()
		if err != nil {
			log.Printf("Error when calling `GetItem` for season item ID `%v`: %v\n", seasonId, err)
			continue
		}

		seasonItem.SetLockData(true)

		_, err = client.ItemUpdateAPI.UpdateItem(ctx, seasonId).BaseItemDto(*seasonItem).Execute()
		if err != nil {
			log.Printf("Error when calling `UpdateItem` for season item ID `%v`: %v\n", seasonId, err)
		}
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
		if tvdbProviderId := item.ProviderIds["Tvdb"]; strings.HasPrefix(tvdbProviderId, "tt") {
			os.Stdout.WriteString(item.GetName() + " " + tvdbProviderId + "\n")
			removeBadTvdbIdAndLock(ctx, client, item.GetId(), adminUserId)
		}
	}
}
