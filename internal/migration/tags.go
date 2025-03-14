package migration

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/spf13/viper"
	"os"
	"strings"
)

type TagDetails struct {
	Name      string `mapstructure:"name" json:"name"`
	StartDate string `mapstructure:"start_date" json:"start_date"`
	EndDate   string `mapstructure:"end_date" json:"end_date"`
}

func (cfg *Config) runZendeskTagsForm() error {
	var tagNames []string
	for _, tag := range cfg.Zendesk.TagsToMigrate {
		tagNames = append(tagNames, tag.Name)
	}

	tagsString := strings.Join(tagNames, ",")

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter Zendesk tags to migrate").
				Placeholder(tagsString).
				Description("Separate tags by commas, and then press Enter").
				Validate(requiredInput).
				Value(&tagsString),
		),
	).WithShowHelp(false).WithKeyMap(customKeyMap()).WithTheme(CustomHuhTheme())

	if err := form.Run(); err != nil {
		return fmt.Errorf("running tag selection form: %w", err)
	}

	tagNames = strings.Split(tagsString, ",")
	for _, tagName := range tagNames {
		if !tagContainsName(cfg.Zendesk.TagsToMigrate, tagName) {
			cfg.Zendesk.TagsToMigrate = append(cfg.Zendesk.TagsToMigrate, TagDetails{Name: tagName})
		}
	}

	viper.Set("zendesk.tags_to_migrate", cfg.Zendesk.TagsToMigrate)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (cfg *Config) runZendeskTagDateForm() error {
	for _, tag := range cfg.Zendesk.TagsToMigrate {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Start date for tag: %s", tag.Name)).
					Description("Use format YYYY-DD-MM (leave blank for no cutoff)").
					Placeholder(tag.StartDate).
					Validate(validDateString).
					Value(&tag.StartDate),
				huh.NewInput().
					Title(fmt.Sprintf("End date for tag: %s", tag.Name)).
					Description("Use format YYYY-DD-MM (leave blank for no cutoff)").
					Placeholder(tag.StartDate).
					Validate(validDateString).
					Value(&tag.EndDate),
			),
		).WithShowHelp(false).WithKeyMap(customKeyMap()).WithTheme(CustomHuhTheme())

		if err := form.Run(); err != nil {
			if errors.As(err, &huh.ErrUserAborted) {
				os.Exit(0)
			}
			return fmt.Errorf("error running date form: %w", err)
		}

		for i, t := range cfg.Zendesk.TagsToMigrate {
			if t.Name == tag.Name {
				cfg.Zendesk.TagsToMigrate[i] = tag
				break
			}
		}
	}

	viper.Set("zendesk.tags_to_migrate", cfg.Zendesk.TagsToMigrate)

	return nil
}

func tagContainsName(d []TagDetails, tagName string) bool {
	for _, tag := range d {
		if tag.Name == tagName {
			return true
		}
	}
	return false
}
