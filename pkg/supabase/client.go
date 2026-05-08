package supabase

import (
	"fmt"
	"mycourse-io-be/constants"

	supabase "github.com/supabase-community/supabase-go"

	"mycourse-io-be/pkg/setting"
)

var Client *supabase.Client

func Setup() error {
	if setting.SupabaseSetting.URL == "" || setting.SupabaseSetting.ServiceRoleKey == "" {
		return fmt.Errorf(constants.MsgSupabaseURLOrServiceKeyRequired)
	}

	client, err := supabase.NewClient(setting.SupabaseSetting.URL, setting.SupabaseSetting.ServiceRoleKey, &supabase.ClientOptions{})
	if err != nil {
		return err
	}

	Client = client
	return nil
}
