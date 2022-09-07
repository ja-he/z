package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"z/internal/cfg"
)

type SyncCommand struct{}

func (c *SyncCommand) Execute(args []string) error {
	errs := []string{}
	msgs := []string{}
	for kID, k := range cfg.GlobalCfg.Ks {
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
	}
	if len(errs) != 0 {
		for _, msg := range msgs {
			fmt.Println(msg)
		}
		return fmt.Errorf("some errors occurred %#v", errs)
	}

	return nil
}
