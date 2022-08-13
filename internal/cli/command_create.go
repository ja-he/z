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

	hasSubdir := blueprint.Subdir != ""
	if !hasSubdir && len(blueprint.Templates) != 1 {
		return fmt.Errorf("blueprint '%s' is NOT in a subdir but also does NOT specify exactly one template", blueprintID)
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
	for file := range filesWithContent {
		if path.IsAbs(file) {
			return fmt.Errorf(
				"This resolved path (%s) appears absolute."+
					"Use paths relative to subdir instead (or to K, if desired and only single file).",
				file,
			)
		}
		fullFilePath := path.Join(k.Path, subdir, file) // if subdir is empty, its fine (just ignored)
		if _, statErr := os.Stat(fullFilePath); !errors.Is(statErr, fs.ErrNotExist) {
			return fmt.Errorf("the file '%s' seems to already exist", file)
		}
		if !hasSubdir {
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
	if hasSubdir {
		err := func() error {
			zDir := path.Join(k.Path, subdir, ".z")
			if err := os.MkdirAll(zDir, 0755); err != nil {
				return err
			}

			if err := os.WriteFile(path.Join(zDir, "open.bash"), []byte(openStr), 0644); err != nil {
				return err
			}

			postScript := "#!/bin/bash\n\n"
			for _, postCmd := range blueprint.Post {
				postScript = postScript + postCmd
			}
			if err := os.WriteFile(path.Join(zDir, "post.bash"), []byte(postScript), 0755); err != nil {
				return err
			}

			return nil
		}()

		if err != nil {
			return fmt.Errorf("unable to create z dir (%s)", err.Error())
		}
	}

	open := exec.Command("bash", "-c", fmt.Sprintf("cd '%s' ; %s", path.Join(k.Path, subdir), openStr))
	open.Stdout, open.Stderr, open.Stdin = os.Stdout, os.Stderr, os.Stdin
	runErr := open.Run()
	if runErr != nil {
		return fmt.Errorf("error running open command (%s)", runErr)
	}

	for i := 0; i < len(blueprint.Post); i++ {
		cmd := exec.Command(
			"bash",
			"-c",
			"cd "+path.Join(k.Path, subdir)+" ; "+blueprint.Post[i],
		)
		cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := cmd.Run(); err != nil || cmd.ProcessState.ExitCode() != 0 {
			fmt.Printf("there was an error trying to run post hook %d:\n", i)
			if err != nil {
				fmt.Printf("  > %s\n", err.Error())
			} else {
				fmt.Printf("  > exit code: %d\n", cmd.ProcessState.ExitCode())
			}
			fmt.Println("would you like to [r]e-run the hook, [c]ontinue with other hooks, or [a]bort all hooks?")
			fmt.Print("r/C/a > ")
			var input string
			_, _ = fmt.Scanln(&input)
			switch input {
			case "r":
				i--
				continue
			case "c", "":
				continue
			case "a":
				break
			default:
				log.Error().Str("directive", input).Msg("unknown directive, continuing...")
				continue
			}
		}
	}

	return nil
}
