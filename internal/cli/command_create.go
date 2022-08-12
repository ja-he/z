package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"z/internal/cfg"
)

type CreateCommand struct{}

func (_ *CreateCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("too few arguments")
	} else if len(args) > 3 {
		return fmt.Errorf("too many arguments")
	}

	kID := args[0]
	k, kOK := cfg.GlobalCfg.Ks[kID]
	if !kOK {
		return fmt.Errorf("no such K '%s'", kID)
	}
	name := args[1]
	blueprintID := ""
	if len(args) == 3 {
		blueprintID = args[2]
	}

	var blueprint cfg.Blueprint
	if blueprintID != "" {
		var ok bool
		blueprint, ok = cfg.GlobalCfg.Blueprints[blueprintID]
		if !ok {
			return fmt.Errorf("no such blueprint '%s'", blueprintID)
		}
		if blueprint.Open == "" {
			return fmt.Errorf("apparently the open command is missing from this blueprint")
		}
	}

	dd := cfg.TemplateFiller{
		K:     k,
		Name:  name,
		Today: strings.Split(time.Now().Local().Format(time.RFC3339), "T")[0],
	}

	openTmpl, err := template.New("openStr").Parse(blueprint.Open)
	if err != nil {
		log.Fatal().Err(err).Str("template", blueprint.Open).
			Msg("unable to parse open template")
	}

	var openStr string
	oBuf := bytes.Buffer{}
	if err := openTmpl.Execute(&oBuf, dd); err != nil {
		return fmt.Errorf("could not execute open template (%s)", err.Error())
	}
	openStr = oBuf.String()

	createZDirIfNecessary := func() error { return nil }

	filesWithContent := map[string]string{}
	for filepathTemplate, contentTemplate := range blueprint.Templates {
		f, err := template.New("filepath").Parse(filepathTemplate)
		if err != nil {
			log.Fatal().Err(err).Str("template", filepathTemplate).
				Msg("unable to parse filepath template (key)")
		}
		c, err := template.New("content").Parse(contentTemplate)
		if err != nil {
			log.Fatal().Err(err).Str("template", contentTemplate).
				Msg("unable to parse content template (value)")
		}

		fBuf := bytes.Buffer{}
		if err := f.Execute(&fBuf, dd); err != nil {
			return fmt.Errorf("could not execute filename template (%s)", err.Error())
		}

		cBuf := bytes.Buffer{}
		if err := c.Execute(&cBuf, dd); err != nil {
			return fmt.Errorf("could not execute content template (%s)", err.Error())
		}

		filesWithContent[fBuf.String()] = cBuf.String()
	}

	// sanity-check files before making contents
	for file := range filesWithContent {
		if path.IsAbs(file) {
			log.Fatal().Str("file", file).Str("tip", "use relative paths instead").
				Msg("this resolved path appears absolute")
		}
		fullFilePath := path.Join(k.Path, file)
		if _, statErr := os.Stat(fullFilePath); !errors.Is(statErr, fs.ErrNotExist) {
			return fmt.Errorf("the file '%s' seems to already exist", file)
		}
		if path.Dir(file) == "." {
			if len(filesWithContent) > 1 {
				log.Fatal().
					Str("file", file).
					Str("blueprint", blueprintID).
					Str("tip", "use a subdir or only create one file").
					Msg("multiple files would be created by the blueprint even though " +
						"the path does not use a subdir")
			}
			_, onlyFile := path.Split(file)
			ext := path.Ext(onlyFile)
			if ext == "" {
				return fmt.Errorf("the resolved file path '%s' seems to lack an extension", file)
			}
			nameSansExt := strings.TrimSuffix(onlyFile, ext)
			dir := path.Join(k.Path, nameSansExt)
			if _, statErr := os.Stat(dir); !errors.Is(statErr, fs.ErrNotExist) {
				return fmt.Errorf("it seems a dir '%s' already exists, so not allowing file '%s'", dir, fullFilePath)
			}
			// doesn't hurt to explicitly put this here, but as we checked for len == 1 this ought to do nothing
			break
		} else {
			createZDirIfNecessary = func() error {
				zDir := path.Join(k.Path, path.Dir(file), ".z")
				if err := os.MkdirAll(zDir, 0755); err != nil {
					return err
				}

				if err := os.WriteFile(path.Join(zDir, "open.bash"), []byte(openStr), 0644); err != nil {
					return err
				}

				// TODO: hooks, probably?

				return nil
			}
		}
	}
	for fileRelative, content := range filesWithContent {
		file := path.Join(k.Path, fileRelative)
		dir := path.Dir(file)
		log.Debug().Str("dir", dir).Msg("creating dir with parents, if needed")
		_ = os.MkdirAll(dir, 0755)
		err := os.WriteFile(file, []byte(content), 0644)
		if err != nil {
			log.Error().Err(err).Str("file", file).Msg("could not write file")
		} else {
			log.Info().Str("file", file).Msg("successfully populated file")
		}
	}
	if err := createZDirIfNecessary(); err != nil {
		return fmt.Errorf("unable to create z dir (%s)", err.Error())
	}

	open := exec.Command("bash", "-c", fmt.Sprintf("cd '%s' ; %s", k.Path, openStr))
	open.Stdout, open.Stderr, open.Stdin = os.Stdout, os.Stderr, os.Stdin
	runErr := open.Run()
	if runErr != nil {
		return fmt.Errorf("error running open command (%s)", runErr)
	}

	// TODO: if post-hooks are to be implemented, that would go here.

	return nil
}
