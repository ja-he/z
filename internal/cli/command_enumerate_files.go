package cli

import (
	"fmt"
	"io"
	"os"
	"path"
	"z/internal/cfg"

	"github.com/rs/zerolog/log"
)

type EnumerateFilesCommand struct {
}

func (c *EnumerateFilesCommand) Execute(args []string) error {
  return enumerateFiles(os.Stdout)
}

func enumerateFiles(w io.Writer) error {
	for id, k := range cfg.GlobalCfg.Ks {
		entries, err := os.ReadDir(k.Path)
		if err != nil {
			return fmt.Errorf("unable to read dir '%s' for K '%s'", k.Path, id)
		}
		for i := range entries {
			if entries[i].Name()[0] == '.' {
				continue
			}
			if entries[i].Type().IsDir() {
				dir := entries[i].Name()
				dirEntries, err := os.ReadDir(path.Join(k.Path, dir))
				if err != nil {
					log.Warn().Str("dir", dir).Msg("could not open dir for reading")
				} else {
					hasZ := func() bool {
						for _, e := range dirEntries {
							if e.Name() == ".z" {
								return true
							}
						}
						return false
					}()
					if hasZ {
						_, err := w.Write([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, dir, "Z")))
						if err != nil {
							log.Warn().Err(err).Msg("error writing result")
						}
						z, err := cfg.ReadZ(path.Join(k.Path, dir))
						if err != nil {
							return fmt.Errorf("unable to get z-data from dir (%s)", err.Error())
						}
						for _, source := range z.Sources {
							_, err := w.Write([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, path.Join(dir, source), "S")))
							if err != nil {
								log.Warn().Err(err).Msg("error writing result")
							}
						}
						for _, object := range z.Objects {
							_, err := w.Write([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, path.Join(dir, object), "O")))
							if err != nil {
								log.Warn().Err(err).Msg("error writing result")
							}
						}
					} else {
						for _, e := range dirEntries {
							if e.Name()[0] == '.' {
								continue
							}
							_, err := w.Write([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, path.Join(dir, e.Name()), "F")))
							if err != nil {
								log.Warn().Err(err).Msg("error writing result")
							}
						}
					}
				}
			} else {
				_, err := w.Write([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, entries[i].Name(), "F")))
				if err != nil {
					log.Warn().Err(err).Msg("error writing result")
				}
			}
		}
	}

  return nil
}
