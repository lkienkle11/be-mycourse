package supabase

import (
	"fmt"

	supabase "github.com/supabase-community/supabase-go"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/setting"
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
