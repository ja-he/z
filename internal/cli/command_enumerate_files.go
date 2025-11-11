package cli

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"z/internal/cfg"

	"github.com/rs/zerolog/log"
)

type EnumerateFilesCommand struct {
	K        bool `long:"k" description:"show k name"`
	FileName bool `long:"file-name" description:"show file name"`
	FileType bool `long:"file-type" description:"show file type"`
	FullPath bool `long:"full-path" description:"show full path"`
}

func (c *EnumerateFilesCommand) Execute(_ []string) error {
	return c.enumerateFiles(os.Stdout)
}

func (c *EnumerateFilesCommand) enumerateFiles(w io.Writer) error {
	for id, k := range cfg.GlobalCfg.Ks {
		entries, err := os.ReadDir(k.Path)
		if err != nil {
			return fmt.Errorf("unable to read dir '%s' for K '%s'", k.Path, id)
		}
		partsSep := "\t"

		addEnabled := func(k, fileName, fileType, fullPath string) []string {
			result := []string{}
			if c.K {
				result = append(result, k)
			}
			if c.FileName {
				result = append(result, fileName)
			}
			if c.FileType {
				result = append(result, fileType)
			}
			if c.FullPath {
				result = append(result, fullPath)
			}
			return result
		}
		writeResult := func(b []byte) error {
			var err error
			_, writeErr := w.Write(b)
			if writeErr != nil {
				err = writeErr
			}
			_, writeErr = w.Write([]byte{'\n'})
			if writeErr != nil {
				err = writeErr
			}
			return err
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
						err := writeResult([]byte(
							strings.Join(
								addEnabled(id, dir, "Z", path.Join(k.Path, dir)),
								partsSep,
							),
						))
						if err != nil {
							log.Warn().Err(err).Msg("error writing result")
						}
						z, err := cfg.ReadZ(path.Join(k.Path, dir))
						if err != nil {
							return fmt.Errorf("unable to get z-data from dir (%s)", err.Error())
						}
						for _, source := range z.Sources {
							err := writeResult([]byte(
								strings.Join(
									addEnabled(id, path.Join(dir, source), "S", path.Join(k.Path, dir, source)),
									partsSep,
								),
							))
							if err != nil {
								log.Warn().Err(err).Msg("error writing result")
							}
						}
						for _, object := range z.Objects {
							err := writeResult([]byte(
								strings.Join(
									addEnabled(id, path.Join(dir, object), "O", path.Join(k.Path, dir, object)),
									partsSep,
								),
							))
							if err != nil {
								log.Warn().Err(err).Msg("error writing result")
							}
						}
					} else {
						for _, e := range dirEntries {
							if e.Name()[0] == '.' {
								continue
							}
							err := writeResult([]byte(
								strings.Join(
									addEnabled(id, path.Join(dir, e.Name()), "F", path.Join(k.Path, dir, e.Name())),
									partsSep,
								),
							))
							if err != nil {
								log.Warn().Err(err).Msg("error writing result")
							}
						}
					}
				}
			} else {
				err := writeResult([]byte(
					strings.Join(
						addEnabled(id, entries[i].Name(), "F", path.Join(k.Path, entries[i].Name())),
						partsSep,
					),
				))
				if err != nil {
					log.Warn().Err(err).Msg("error writing result")
				}
			}
		}
	}

	return nil
}
