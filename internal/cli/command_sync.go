package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"time"

	"z/internal/cfg"

	"github.com/rs/zerolog/log"
)

// SyncCommand is the command that syncs all Ks.
type SyncCommand struct{}

// Execute runs the sync command.
func (c *SyncCommand) Execute(_ []string) error {
	errs := []string{}
	msgs := []string{}
	for kID, k := range cfg.GlobalCfg.Ks {

		// skip manually synced Ks
		if k.URL == "" {
			log.Info().Msgf("skipping K '%s' (manual sync)", kID)
			continue
		}
		log.Info().Msgf("syncing K '%s' (auto sync)", kID)

		if hadToInitialize := ensureInitialized(kID, k); !hadToInitialize {

			fmt.Println("updating", kID)
			cmd := exec.Command(
				"bash", "-c",
				fmt.Sprintf(`
				cd "%s"
				local_update=false
				remote_update=false
				if [[ $(git status --porcelain) ]]; then
				  local_update=true
				fi
				git fetch
				local_head=$(git rev-parse @)
				remote_head=$(git rev-parse @{u})
				if [[ "${local_head}" != "${remote_head}" ]]; then
				  remote_update=true
				fi
				
				if [ "${local_update}" == "true" ]; then
				  git add .
				  git commit -m "%s Update"
				fi
				can_push=true
				if [ "${remote_update}" == "true" ]; then
				  git pull --rebase || can_push=false
				fi
				if [ "${local_update}" == "true" ]; then
				  if [ "${can_push}" == "true" ]; then
				    git push
				  else
				    echo "WARN: cannot push update in %s yet!"
						exit 1
				  fi
				fi`,
					k.Path, strings.Split(time.Now().Local().Format(time.RFC3339), "T")[0], kID,
				),
			)
			cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin

			if err := cmd.Run(); err != nil {
				errs = append(errs, err.Error())
				msgs = append(
					msgs,
					fmt.Sprintf(
						"%s could not be synced!\nDo `cd '%s'` and resolve it there.\n",
						kID,
						k.Path,
					),
				)
			}
		} else {
			log.Info().Str("K", kID).Msg("as K was just cloned, skipped pull/push for it")
		}
	}
	if len(errs) != 0 {
		for _, msg := range msgs {
			fmt.Println(msg)
		}
		return fmt.Errorf("some errors occurred %#v", errs)
	}

	return nil
}

func ensureInitialized(kID string, k cfg.K) (initialized bool) {
	_, err := os.Stat(k.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Info().Str("K", kID).Str("path", k.Path).Msg("K's path does not exist, initializing it by git clone")
			clone := exec.Command("git", "clone", k.URL, k.Path)
			clone.Stdout, clone.Stderr, clone.Stdin = os.Stdout, os.Stderr, os.Stdin
			err := clone.Run()
			if err != nil {
				log.Error().
					Err(err).
					Strs("args", clone.Args).
					Str("K", kID).
					Msg("error executing clone command")
			}
			return true
		} else {
			log.Fatal().Str("path", k.Path).Msg("stat err for K path but it appears to exist?")
		}
	}
	return false
}
