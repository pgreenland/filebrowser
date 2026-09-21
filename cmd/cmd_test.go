package cmd

import (
	"path/filepath"
	"testing"

	"github.com/asdine/storm/v3"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/filebrowser/filebrowser/v2/auth"
	"github.com/filebrowser/filebrowser/v2/settings"
	"github.com/filebrowser/filebrowser/v2/storage/bolt"
)

// TestEnvCollisions ensures that there are no collisions in the produced environment
// variable names for all commands and their flags.
func TestEnvCollisions(t *testing.T) {
	testEnvCollisions(t, rootCmd)
}

func testEnvCollisions(t *testing.T, cmd *cobra.Command) {
	for _, cmd := range cmd.Commands() {
		testEnvCollisions(t, cmd)
	}

	replacements := generateEnvKeyReplacements(cmd)
	envVariables := []string{}

	for i := range replacements {
		if i%2 != 0 {
			envVariables = append(envVariables, replacements[i])
		}
	}

	duplicates := lo.FindDuplicates(envVariables)

	if len(duplicates) > 0 {
		t.Errorf("Found duplicate environment variable keys for command %q: %v", cmd.Name(), duplicates)
	}
}

// TestGetSettingsFollowExternalSymlinks ensures that the followExternalSymlinks
// flag is persisted to the server config when set via "config set".
func TestGetSettingsFollowExternalSymlinks(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	addConfigFlags(flags)

	if err := flags.Parse([]string{"--followExternalSymlinks"}); err != nil {
		t.Fatal(err)
	}

	set := &settings.Settings{AuthMethod: auth.MethodJSONAuth}
	ser := &settings.Server{}

	if _, err := getSettings(flags, set, ser, &auth.JSONAuth{}, false); err != nil {
		t.Fatal(err)
	}

	if !ser.FollowExternalSymlinks {
		t.Error("expected FollowExternalSymlinks to be persisted as true")
	}
}

func TestRotateSigningKeyOnBoot(t *testing.T) {
	db, err := storm.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	st, err := bolt.NewStorage(db)
	if err != nil {
		t.Fatal(err)
	}

	initial := []byte("initial-signing-key")
	if err := st.Settings.Save(&settings.Settings{Key: initial}); err != nil {
		t.Fatal(err)
	}

	v := viper.New()
	v.Set("rotateSigningKeyOnBoot", true)

	if err := rotateSigningKeyOnBoot(v, st); err != nil {
		t.Fatal(err)
	}

	updated, err := st.Settings.Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Key) == 0 {
		t.Fatal("expected non-empty key after rotation")
	}
	if string(updated.Key) == string(initial) {
		t.Fatal("expected key to be rotated")
	}
}
