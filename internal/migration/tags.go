package migration

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/spf13/viper"
	"os"
)

type TagDetails struct {
	Name      string `mapstructure:"name" json:"name"`
	StartDate string `mapstructure:"start_date" json:"start_date"`
	EndDate   string `mapstructure:"end_date" json:"end_date"`
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
