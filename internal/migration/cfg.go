package migration

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const (
	configFileSubPath = "/migrator_config.json"
)

var (
	CfgFile string
)

type Config struct {
	Zendesk ZdCfg `mapstructure:"zendesk" json:"zendesk"`
	CW      CwCfg `mapstructure:"connectwise_psa" json:"connectwise_psa"`
}

type ZdCfg struct {
	Creds         zendesk.Creds `mapstructure:"api_creds" json:"api_creds"`
	TagsToMigrate []string      `mapstructure:"tags_to_migrate" json:"tags_to_migrate"`
	FieldIds      ZdFieldIds    `mapstructure:"field_ids" json:"field_ids"`
}

type CwCfg struct {
	Creds              psa.Creds  `mapstructure:"api_creds" json:"api_creds"`
	ClosedStatusId     int        `mapstructure:"closed_status_id" json:"closed_status_id"`
	OpenStatusId       int        `mapstructure:"open_status_id" json:"open_status_id"`
	DestinationBoardId int        `mapstructure:"destination_board_id" json:"destination_board_id"`
	FieldIds           CwFieldIds `mapstructure:"field_ids" json:"field_ids"`
}

type ZdFieldIds struct {
	PsaCompanyId int `mapstructure:"psa_company_id" json:"psa_company_id"`
	PsaContactId int `mapstructure:"psa_contact_id" json:"psa_contact_id"`
}

type CwFieldIds struct {
	ZendeskTicketId int `mapstructure:"zendesk_ticket_id" json:"zendesk_ticket_id"`
}

// InitConfig reads in config file and ENV variables if set.
func InitConfig() error {
	// Find home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("error getting home directory", "error", err)
		return fmt.Errorf("getting home directory: %w", err)
	}

	if CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(CfgFile)
	} else {
		// Search config in home directory with name ".zendesk-connectwise-migrator" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("json")
		viper.SetConfigName("migrator_config")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			setCfgDefaults()
			path := home + configFileSubPath
			fmt.Println("Creating default config file")
			if err := viper.WriteConfigAs(path); err != nil {
				slog.Error("error creating default config file", "error", err)
				fmt.Println("Error creating default config file:", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Error reading config file:", err)
			os.Exit(1)
		}
	}

	return nil
}

func (cfg *Config) ValidateCreds() error {
	slog.Debug("validating required fields")
	var missing []string

	requiredFields := map[string]string{
		"zendesk.api_creds.token":               cfg.Zendesk.Creds.Token,
		"zendesk.api_creds.username":            cfg.Zendesk.Creds.Username,
		"zendesk.api_creds.subdomain":           cfg.Zendesk.Creds.Subdomain,
		"connectwise_psa.api_creds.company_id":  cfg.CW.Creds.CompanyId,
		"connectwise_psa.api_creds.public_key":  cfg.CW.Creds.PublicKey,
		"connectwise_psa.api_creds.private_key": cfg.CW.Creds.PrivateKey,
		"connectwise_psa.api_creds.client_id":   cfg.CW.Creds.ClientId,
	}

	for k, v := range requiredFields {
		if v == "" {
			slog.Warn("missing required config value", "key", k)
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		slog.Error("missing required config values", "missing", missing)
		return errors.New("missing 1 or more required config values")
	}

	return nil
}

func (cfg *Config) ValidateAndPrompt() error {
	if err := cfg.ValidateCreds(); err != nil {
		if err := cfg.RunCredsForm(); err != nil {
			return fmt.Errorf("error running creds form: %w", err)
		}
	}

	if err := cfg.ValidateZendeskTags(); err != nil {
		if err := cfg.RunZendeskTagsForm(); err != nil {
			return fmt.Errorf("error validating zendesk tags: %w", err)
		}
	}

	//if err := cfg.ValidateZendeskCustomFields(); err != nil {
	//	// TODO: add form field for this
	//	return fmt.Errorf("error validating zendesk custom fields: %w", err)
	//}

	if err := cfg.ValidateConnectWiseCustomField(); err != nil {
		if err := cfg.RunConnectwiseFieldForm(); err != nil {
			return fmt.Errorf("error validating connectwise custom fields: %w", err)
		}
		return fmt.Errorf("error validating connectwise custom fields: %w", err)
	}

	return nil
}

func (cfg *Config) ValidateZendeskTags() error {
	if len(cfg.Zendesk.TagsToMigrate) == 0 {
		slog.Warn("no tags selected to migrate")
		return errors.New("no tags selected to migrate")
	}

	return nil
}

func (cfg *Config) ValidateZendeskCustomFields() error {
	if cfg.Zendesk.FieldIds.PsaCompanyId == 0 || cfg.Zendesk.FieldIds.PsaContactId == 0 {
		slog.Warn("no Zendesk custom field IDs set")
		return errors.New("no Zendesk custom field IDs set")
	}

	return nil
}

func (cfg *Config) ValidateConnectWiseCustomField() error {
	if cfg.CW.FieldIds.ZendeskTicketId == 0 {
		slog.Warn("no ConnectWise PSA custom field ID set")
		return errors.New("no ConnectWise PSA custom field ID set")
	}

	return nil
}

func (cfg *Config) RunCredsForm() error {
	if err := cfg.credsForm().Run(); err != nil {
		slog.Error("error running creds form", "error", err)
		return fmt.Errorf("running creds form: %w", err)
	}
	slog.Debug("creds form completed", "cfg", cfg)

	viper.Set("zendesk", cfg.Zendesk)
	viper.Set("connectwise_psa", cfg.CW)
	if err := viper.WriteConfig(); err != nil {
		slog.Error("error writing config file", "error", err)
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (cfg *Config) credsForm() *huh.Form {
	return huh.NewForm(
		inputGroup("Zendesk Token", &cfg.Zendesk.Creds.Token, requiredInput, true),
		inputGroup("Zendesk Username", &cfg.Zendesk.Creds.Username, requiredInput, true),
		inputGroup("Zendesk Subdomain", &cfg.Zendesk.Creds.Subdomain, requiredInput, true),
		inputGroup("ConnectWise Company ID", &cfg.CW.Creds.CompanyId, requiredInput, true),
		inputGroup("ConnectWise Public Key", &cfg.CW.Creds.PublicKey, requiredInput, true),
		inputGroup("ConnectWise Private Key", &cfg.CW.Creds.PrivateKey, requiredInput, true),
		inputGroup("ConnectWise Client ID", &cfg.CW.Creds.ClientId, requiredInput, true),
	).WithHeight(3).WithShowHelp(false).WithTheme(huh.ThemeBase())
}

// inputGroup creates a huh Group with an input field, this is just to make cfg.credsForm prettier.
func inputGroup(title string, value *string, validate func(string) error, inline bool) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title(title).
			Placeholder(*value).
			Validate(validate).
			Inline(inline).
			Value(value),
	)
}

func (cfg *Config) RunZendeskTagsForm() error {
	var tagsString string
	input := huh.NewInput().
		Title("Enter Zendesk Tags to Migrate").
		Description("Separate tags by commas, and then press Enter").
		Value(&tagsString)

	if err := input.WithTheme(huh.ThemeBase()).Run(); err != nil {
		return fmt.Errorf("running tag selection form: %w", err)
	}

	// Split tags by comma, and then trim any whitespace from each tag
	var tags []string
	if tagsString != "" {
		tags = strings.Split(tagsString, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
	}

	viper.Set("zendesk.tags_to_migrate", tags)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (cfg *Config) RunConnectwiseFieldForm() error {
	var s string
	input := huh.NewInput().
		Title("Enter ConnectWise PSA Custom Field ID").
		Description("Navigate to ConnectWise PSA > System > Setup Tables > Custom Fields > Ticket\n" +
			"If you haven't made one, create a new Custom Field with the name 'Zendesk Ticket ID'\n" +
			"Field Type: Number, Number of Decimals: 0. Save the field and enter the ID here.").
		Value(&s)

	if err := input.WithTheme(huh.ThemeBase()).Run(); err != nil {
		return fmt.Errorf("running custom field form: %w", err)
	}

	i, err := strToInt(s)
	if err != nil {
		return err
	}

	viper.Set("connectwise_psa.field_ids.zendesk_ticket_id", i)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func strToInt(s string) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		slog.Error("error converting string to int", "error", err)
		return 0, fmt.Errorf("converting string to int: %w", err)
	}

	return i, nil
}

// Validator for required huh Input fields
func requiredInput(s string) error {
	if s == "" {
		return errors.New("field is required")
	}
	return nil
}

func setCfgDefaults() {
	slog.Debug("setting config defaults")
	viper.SetDefault("zendesk", ZdCfg{})
	viper.SetDefault("connectwise_psa", CwCfg{})
}
