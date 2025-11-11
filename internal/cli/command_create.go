package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"z/internal/cfg"
)

type CreateCommand struct {
	Args struct {
		K         string `positional-arg-name:"K" required:"yes" description:"ID of the knowledge base (K) to create in"`
		Name      string `positional-arg-name:"name" required:"yes" description:"Name for the new note/file"`
		Blueprint string `positional-arg-name:"blueprint" description:"Blueprint to use for the note"`
	} `positional-args:"yes"`
}

func (c *CreateCommand) Execute(_ []string) error {
	kID := c.Args.K
	k, kOK := cfg.GlobalCfg.Ks[kID]
	if !kOK {
		available := make([]string, 0, len(cfg.GlobalCfg.Ks))
		for id := range cfg.GlobalCfg.Ks {
			available = append(available, id)
		}
		return fmt.Errorf("no such K '%s'\nAvailable Ks: %s", kID, strings.Join(available, ", "))
	}
	name := c.Args.Name
	blueprintID := c.Args.Blueprint

	var blueprint cfg.Blueprint
	if blueprintID != "" {
		var ok bool
		blueprint, ok = cfg.GlobalCfg.Blueprints[blueprintID]
		if !ok {
			available := make([]string, 0, len(cfg.GlobalCfg.Blueprints))
			for id := range cfg.GlobalCfg.Blueprints {
				available = append(available, id)
			}
			if len(available) > 0 {
				return fmt.Errorf("no such blueprint '%s'\nAvailable blueprints: %s", blueprintID, strings.Join(available, ", "))
			}
			return fmt.Errorf("no such blueprint '%s' (no blueprints configured)", blueprintID)
		}
		if blueprint.Open == "" {
			return fmt.Errorf("blueprint '%s' is invalid: missing required 'open' command", blueprintID)
		}
	}

	dd := cfg.TemplateFiller{
		K:     k,
		Name:  name,
		Today: strings.Split(time.Now().Local().Format(time.RFC3339), "T")[0],
		Now:   time.Now().Local().Format(time.RFC3339),
	}

	fillTemplate := func(t string) (string, error) {
		tmpl, err := template.New("tmpl").Parse(t)
		if err != nil {
			return "", fmt.Errorf("unable to parse template '%s' (%s)", t, err.Error())
		}

		b := bytes.Buffer{}
		if err := tmpl.Execute(&b, dd); err != nil {
			return "", fmt.Errorf("could not execute template (%s)", err.Error())
		}
		return b.String(), nil
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

	viewTmpl, err := template.New("viewStr").Parse(blueprint.View)
	if err != nil {
		log.Fatal().Err(err).Str("template", blueprint.View).
			Msg("unable to parse view template")
	}
	var viewStr string
	vBuf := bytes.Buffer{}
	if err := viewTmpl.Execute(&vBuf, dd); err != nil {
		return fmt.Errorf("could not execute open template (%s)", err.Error())
	}
	viewStr = vBuf.String()

	hasSubdir := blueprint.Subdir != ""
	if !hasSubdir {
		if len(blueprint.Templates) != 1 {
			return fmt.Errorf("blueprint '%s' is NOT in a subdir but also does NOT specify exactly one template", blueprintID)
		}
		if len(blueprint.Post) > 0 {
			return fmt.Errorf("blueprint '%s' is NOT in a subdir but still has post hooks", blueprintID)
		}
	}
	subdir, subdirTmplErr := func() (string, error) {
		tmpl, err := template.New("subdir").Parse(blueprint.Subdir)
		if err != nil {
			return "", fmt.Errorf("unable to parse subdir template")
		}
		buf := bytes.Buffer{}
		if err := tmpl.Execute(&buf, dd); err != nil {
			return "", fmt.Errorf("could not execute filename template (%s)", err.Error())
		}
		return buf.String(), nil
	}()
	if subdirTmplErr != nil {
		return subdirTmplErr
	}

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
	onlyFileIfNoSubdir := func() string {
		log.Fatal().Msg("Erroneously called onlyFileIfNoSubdir() while having a subdir?!")
		return ""
	}
	for file := range filesWithContent {
		if path.IsAbs(file) {
			return fmt.Errorf(
				"this resolved path (%s) appears absolute."+
					"Use paths relative to subdir instead (or to K, if desired and only single file)",
				file,
			)
		}
		fullFilePath := path.Join(k.Path, subdir, file) // if subdir is empty, its fine (just ignored)
		if _, statErr := os.Stat(fullFilePath); !errors.Is(statErr, fs.ErrNotExist) {
			return fmt.Errorf("the file '%s' seems to already exist", file)
		}
		if !hasSubdir {
			_, onlyFile := path.Split(file)
			onlyFileIfNoSubdir = func() string { return onlyFile }
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
		}
	}
	for fileRelative, content := range filesWithContent {
		file := path.Join(k.Path, subdir, fileRelative)
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

	// create .z dir in subdir, if there is a subdir
	pFilled := make([]string, len(blueprint.Post))
	if hasSubdir {
		err := func() error {
			zDir := path.Join(k.Path, subdir, ".z")
			if err := os.MkdirAll(zDir, 0755); err != nil {
				return err
			}

			for i := range blueprint.Post {
				var err error
				pFilled[i], err = fillTemplate(blueprint.Post[i])
				if err != nil {
					return err
				}
			}
			sFilled := make([]string, len(blueprint.Sources))
			for i := range blueprint.Sources {
				var err error
				sFilled[i], err = fillTemplate(blueprint.Sources[i])
				if err != nil {
					return err
				}
			}
			oFilled := make([]string, len(blueprint.Objects))
			for i := range blueprint.Objects {
				var err error
				oFilled[i], err = fillTemplate(blueprint.Objects[i])
				if err != nil {
					return err
				}
			}
			zYAML, marshalErr := yaml.Marshal(cfg.Z{
				Open:    openStr,
				View:    viewStr,
				Post:    pFilled,
				Sources: sFilled,
				Objects: oFilled,
			})
			if marshalErr != nil {
				return fmt.Errorf("unable to marshal z yaml (%s)", marshalErr.Error())
			}
			if err := os.WriteFile(path.Join(zDir, "z.yml"), zYAML, 0644); err != nil {
				return fmt.Errorf("error writing '.z/z.yml' (%s)", err.Error())
			}

			return nil
		}()

		if err != nil {
			return fmt.Errorf("unable to create z dir (%s)", err.Error())
		}
	}

	// run the open command
	openCmd := &OpenCommand{}
	openCmd.Args.K = kID
	openCmd.Args.File = func() string {
		if hasSubdir {
			return subdir
		} else {
			return onlyFileIfNoSubdir()
		}
	}()
	openCmd.Args.Type = func() string {
		if hasSubdir {
			return "Z"
		} else {
			return "F"
		}
	}()
	return openCmd.Execute(nil)
}
