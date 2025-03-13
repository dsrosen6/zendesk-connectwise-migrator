package migration

import (
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/spf13/viper"
	"log/slog"
	"strings"
)

type TagDetails struct {
	Name      string `mapstructure:"name" json:"name"`
	StartDate string `mapstructure:"start_date" json:"start_date"`
	EndDate   string `mapstructure:"end_date" json:"end_date"`
}

func (cfg *Config) validateZendeskTags() error {
	if len(cfg.Zendesk.TagsToMigrate) == 0 {
		if err := cfg.runZendeskTagsForm(); err != nil {
			return fmt.Errorf("error running tag entry form: %w", err)
		}
	}

	for _, tag := range cfg.Zendesk.TagsToMigrate {
		if err := cfg.validateTagDates(&tag); err != nil {
			if err := cfg.runZendeskTagDateForm(&tag); err != nil {
				return fmt.Errorf("error running tag date form for tag %s: %w", tag.Name, err)
			}
		}
	}

	return nil
}

func (cfg *Config) runZendeskTagsForm() error {
	var tagNames []string
	for _, tag := range cfg.Zendesk.TagsToMigrate {
		tagNames = append(tagNames, tag.Name)
	}

	tagsString := strings.Join(tagNames, ",")

	input := huh.NewInput().
		Title("Enter Zendesk tags to migrate").
		Placeholder(tagsString).
		Description("Separate tags by commas, and then press Enter").
		Validate(requiredInput).
		Value(&tagsString).
		WithTheme(huh.ThemeBase16())

	if err := input.Run(); err != nil {
		return fmt.Errorf("running tag selection form: %w", err)
	}

	tagNames = strings.Split(tagsString, ",")
	for _, tagName := range tagNames {
		if !contains(cfg.Zendesk.TagsToMigrate, tagName) {
			cfg.Zendesk.TagsToMigrate = append(cfg.Zendesk.TagsToMigrate, TagDetails{Name: tagName})
		}
	}

	viper.Set("zendesk.tags_to_migrate", cfg.Zendesk.TagsToMigrate)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func contains(d []TagDetails, tagName string) bool {
	for _, tag := range d {
		if tag.Name == tagName {
			return true
		}
	}
	return false
}

func (cfg *Config) validateTagDates(tag *TagDetails) error {
	if err := validDateString(tag.StartDate); err != nil {
		tag.StartDate = ""
		slog.Warn("invalid zendesk start date string", "tag", tag.Name)
		return fmt.Errorf("invalid zendesk start date string for tag %s", tag.Name)
	}

	if err := validDateString(tag.EndDate); err != nil {
		tag.StartDate = ""
		slog.Warn("invalid zendesk end date string", "tag", tag.Name)
		return fmt.Errorf("invalid zendesk end date string for tag %s", tag.Name)
	}

	return nil
}

func (cfg *Config) runZendeskTagDateForm(tag *TagDetails) error {
	title := fmt.Sprintf("Begin date to look for orgs with Zendesk tag %s", tag.Name)
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description("Use format YYYY-DD-MM (leave blank for no cutoff)").
				Placeholder(tag.StartDate).
				Validate(validDateString).
				Value(&tag.StartDate),
			huh.NewInput().
				Title("End date to look for Zendesk tickets").
				Description("Use format YYYY-DD-MM (leave blank for no cutoff)").
				Placeholder(tag.StartDate).
				Validate(validDateString).
				Value(&tag.EndDate),
		),
	).WithShowHelp(false).WithTheme(huh.ThemeBase16())

	if err := form.Run(); err != nil {
		return fmt.Errorf("error running date form: %w", err)
	}

	viper.Set("zendesk.tags_to_migrate", cfg.Zendesk.TagsToMigrate)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
