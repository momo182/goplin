package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/imroc/req/v3"
	"github.com/piccobit/goplin"
	"github.com/spf13/viper"
)

type CliContext struct {
	Debug bool
}

type ListTagsCmd struct {
	DuplicatesOnly bool     `name:"duplicates-only" help:"List only duplicate tags."`
	IDs            []string `arg:"" optional:"" name:"id" help:"List tags with the specified IDs."`
	OrderBy        string   `name:"order-by" help:"Order by specified field."`
	OrderDir       string   `name:"order-dir" help:"Order by specified direction: ASC or DESC."`
}

type ListNotesCmd struct {
	By       string   `optional:"" name:"by" help:"Find by ID or tag."`
	IDs      []string `arg:"" optional:"" name:"id" help:"List notes with the specified IDs or tag IDs."`
	OrderBy  string   `name:"order-by" help:"Order by specified field."`
	OrderDir string   `name:"order-dir" help:"Order by specified direction: ASC or DESC."`
}

type DeleteTagsCmd struct {
	IDs []string `arg:"" name:"id" help:"Delete tags with the specified IDs."`
}

var cli struct {
	Debug bool `help:"Enable debug mode."`

	List struct {
		Tags  ListTagsCmd  `cmd:"" requires:"" help:"List tags."`
		Notes ListNotesCmd `cmd:"" requires:"" help:"List notes."`
	} `cmd:"" help:"Joplin list commands."`

	Delete struct {
		Tags DeleteTagsCmd `cmd:"" requires:"" help:"Delete tags."`
	} `cmd:"" help:"Joplin delete commands."`
}

var (
	client *goplin.Client
)

func getItemTypes() []string {
	return []string{
		"unknown",
		"note",
		"folder",
		"setting",
		"resource",
		"tag",
		"note_tag",
		"search",
		"alarm",
		"master_key",
		"item_change",
		"note_resource",
		"resource_local_state",
		"revision",
		"migration",
		"smart_filter",
		"command",
	}
}

func (ltc *ListTagsCmd) Run(ctx *CliContext) error {
	const ListTagsFormat = "%-32s \u2502 %-32s \u2502 %s\n"
	if ctx.Debug {
		req.EnableDumpAll()
		req.EnableDebugLog()
	}

	if !ltc.DuplicatesOnly {
		fmt.Println("Tags:")
		fmt.Printf(ListTagsFormat, "ID", "Parent ID", "Title")
	}

	if len(ltc.IDs) == 0 {
		tags, err := client.GetAllTags(ltc.OrderBy, ltc.OrderDir)
		if err != nil {
			return err
		}

		if ltc.DuplicatesOnly {
			fmt.Println("Duplicate tags:")

			tagsFound := make(map[string][]string)

			for _, tag := range tags {
				tagsFound[tag.Title] = append(tagsFound[tag.Title], tag.ID)
			}

			for title, ids := range tagsFound {
				if len(ids) > 1 {
					fmt.Printf("%s:", title)
					for _, id := range ids {
						fmt.Printf(" %s", id)
					}
					fmt.Println()
				}
			}
		} else {
			for _, tag := range tags {
				fmt.Printf(ListTagsFormat, tag.ID, tag.ParentID, tag.Title)
			}
		}
	} else {
		for _, id := range ltc.IDs {
			tag, err := client.GetTag(id)
			if err != nil {
				fmt.Printf(ListTagsFormat, id, "ERROR: tag not found", "")
			} else {
				fmt.Printf(ListTagsFormat, tag.ID, tag.ParentID, tag.Title)
			}
		}
	}

	return nil
}

func (lnc *ListNotesCmd) Run(ctx *CliContext) error {
	const ListNotesFormat = "%-32s \u2502 %-32s \u2502 %s\n"
	if ctx.Debug {
		req.EnableDumpAll()
		req.EnableDebugLog()
	}

	fmt.Println("Notes:")
	fmt.Printf(ListNotesFormat, "ID", "Parent ID", "Title")

	if len(lnc.IDs) == 0 {
		notes, err := client.GetAllNotes(lnc.OrderBy, lnc.OrderDir)
		if err != nil {
			return err
		}

		for _, note := range notes {
			fmt.Printf(ListNotesFormat, note.ID, note.ParentID, note.Title)
		}
	} else {
		if strings.ToLower(lnc.By) == "tag" {
			for _, id := range lnc.IDs {
				notes, err := client.GetNotes(id, lnc.OrderBy, lnc.OrderDir)
				if err != nil {
					fmt.Printf(ListNotesFormat, id, "ERROR: note not found", "")
				} else {
					for _, note := range notes {
						fmt.Printf(ListNotesFormat, note.ID, note.ParentID, note.Title)
					}
				}
			}
		} else {
			for _, id := range lnc.IDs {
				note, err := client.GetNote(id)
				if err != nil {
					fmt.Printf(ListNotesFormat, id, "ERROR: note not found", "")
				} else {
					fmt.Printf(ListNotesFormat, note.ID, note.ParentID, note.Title)
				}

			}
		}
	}

	return nil
}

func (dtc *DeleteTagsCmd) Run(ctx *CliContext) error {
	if ctx.Debug {
		req.EnableDumpAll()
		req.EnableDebugLog()
	}

	for _, id := range dtc.IDs {
		err := client.DeleteTag(id)
		if err != nil {
			fmt.Printf("Could not find tag with ID '%s'\n", id)
		} else {
			fmt.Printf("Tag with ID '%s' deleted'\n", id)
		}
	}

	return nil
}

func main() {
	var err error

	viper.SetDefault("api_token", "")
	viper.SetConfigName(".goplin") // name of config file (without extension)
	viper.SetConfigType("yaml")    // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("$HOME")   // call multiple times to add many search paths
	err = viper.ReadInConfig()     // Find and read the config file
	if err != nil {                // handle errors reading the config file
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
		} else {
			// Config file was found but another error was produced
			panic(fmt.Errorf("fatal error config file: %w", err))
		}
	}

	apiToken := viper.GetString("api_token")

	client, err = goplin.New(apiToken)
	if err != nil {
		log.Fatal(err)
	}

	if len(apiToken) == 0 {
		viper.Set("api_token", client.GetApiToken())
		err = viper.WriteConfigAs(path.Join(os.Getenv("HOME"), ".goplin"))
		if err != nil {
			log.Fatal(err)
		}
	}

	ctx := kong.Parse(&cli)
	err = ctx.Run(&CliContext{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
