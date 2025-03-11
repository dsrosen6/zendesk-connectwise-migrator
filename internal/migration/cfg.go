package migration

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"sort"
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
	PsaCompanyId int64 `mapstructure:"psa_company_id" json:"psa_company_id"`
	PsaContactId int64 `mapstructure:"psa_contact_id" json:"psa_contact_id"`
}

type CwFieldIds struct {
	ZendeskTicketId int `mapstructure:"zendesk_ticket_id" json:"zendesk_ticket_id"`
}

// InitConfig reads in config file and ENV variables if set.
func InitConfig() (*Config, error) {
	// Find home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("error getting home directory", "error", err)
		return nil, fmt.Errorf("getting home directory: %w", err)
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

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		slog.Error("unmarshaling config", "error", err)
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return cfg, nil
}

func (cfg *Config) ValidateAndPrompt() error {
	if err := cfg.validateCreds(); err != nil {
		if err := cfg.runCredsForm(); err != nil {
			return fmt.Errorf("error running creds form: %w", err)
		}
	}

	if err := cfg.validateZendeskTags(); err != nil {
		if err := cfg.runZendeskTagsForm(); err != nil {
			return fmt.Errorf("error validating zendesk tags: %w", err)
		}
	}

	if err := cfg.validateConnectwiseCustomField(); err != nil {
		if err := cfg.runConnectwiseFieldForm(); err != nil {
			return fmt.Errorf("error validating connectwise custom fields: %w", err)
		}
	}

	return nil
}

func (cfg *Config) PromptAllFields() error {
	if err := cfg.runCredsForm(); err != nil {
		slog.Error("error running creds form", "error", err)
	}

	if err := cfg.runZendeskTagsForm(); err != nil {
		slog.Error("error running tags form", "error", err)
	}

	if err := cfg.runConnectwiseFieldForm(); err != nil {
		slog.Error("error running field form", "error", err)
	}

	return nil
}

func (cfg *Config) validateCreds() error {
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

func (cfg *Config) validateZendeskTags() error {
	if len(cfg.Zendesk.TagsToMigrate) == 0 {
		slog.Warn("no tags selected to migrate")
		return errors.New("no tags selected to migrate")
	}

	return nil
}

func (cfg *Config) validateConnectwiseCustomField() error {
	if cfg.CW.FieldIds.ZendeskTicketId == 0 {
		slog.Warn("no ConnectWise PSA custom field ID set")
		return errors.New("no ConnectWise PSA custom field ID set")
	}

	slog.Debug("connectwise custom field id found in config", "zendeskTicketId", cfg.CW.FieldIds.ZendeskTicketId)
	return nil
}

func (cfg *Config) validateZendeskCustomFields() error {
	if cfg.Zendesk.FieldIds.PsaCompanyId == 0 || cfg.Zendesk.FieldIds.PsaContactId == 0 {
		slog.Warn("no Zendesk custom field IDs set")
		return errors.New("no Zendesk custom field IDs set")
	}

	slog.Debug("zendesk custom field ids in config",
		"psaCompanyId", cfg.Zendesk.FieldIds.PsaContactId,
		"psaContactId", cfg.Zendesk.FieldIds.PsaContactId,
	)
	return nil
}

func (cfg *Config) validateConnectwiseBoardId() error {
	if cfg.CW.DestinationBoardId == 0 {
		slog.Warn("no destination board ID set")
		return errors.New("no destination board ID set")
	}

	slog.Debug("connectwise board id found in config", "boardId", cfg.CW.DestinationBoardId)
	return nil
}

func (cfg *Config) validateConnectwiseStatuses() error {
	if cfg.CW.OpenStatusId == 0 || cfg.CW.ClosedStatusId == 0 {
		slog.Warn("no open status ID or closed status ID set")
		return errors.New("no open status ID or closed status ID set")
	}

	slog.Debug("board status ids", "open", cfg.CW.OpenStatusId, "closed", cfg.CW.ClosedStatusId)
	return nil
}

func (cfg *Config) runCredsForm() error {
	if err := cfg.credsForm().Run(); err != nil {
		slog.Error("error running creds form", "error", err)
		return fmt.Errorf("running creds form: %w", err)
	}
	slog.Debug("creds form completed", "cfg", cfg)

	viper.Set("zendesk.api_creds", cfg.Zendesk.Creds)
	viper.Set("connectwise_psa.api_creds", cfg.CW.Creds)
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
	).WithShowHelp(false).WithTheme(huh.ThemeBase16())
}

func (cfg *Config) runZendeskTagsForm() error {
	tagsString := strings.Join(cfg.Zendesk.TagsToMigrate, ",")
	input := huh.NewInput().
		Title("Enter Zendesk Tags to Migrate").
		Placeholder(tagsString).
		Description("Separate tags by commas, and then press Enter").
		Validate(requiredInput).
		Value(&tagsString).
		WithTheme(huh.ThemeBase16())

	if err := input.Run(); err != nil {
		return fmt.Errorf("running tag selection form: %w", err)
	}

	// Split tags by comma, and then trim any whitespace from each tag
	var tags []string

	tags = strings.Split(tagsString, ",")
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}

	viper.Set("zendesk.tags_to_migrate", tags)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (cfg *Config) runConnectwiseFieldForm() error {
	s := strconv.Itoa(cfg.CW.FieldIds.ZendeskTicketId)
	input := huh.NewInput().
		Title("Enter ConnectWise PSA Custom Field ID").
		Description("See docs if you have not made one.").
		Placeholder(s).
		Validate(requiredInput).
		Value(&s).
		WithTheme(huh.ThemeBase16())

	if err := input.Run(); err != nil {
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

func (c *Client) processZendeskPsaForms(ctx context.Context) error {
	uf, err := c.ZendeskClient.GetUserFieldByKey(ctx, psaContactFieldKey)
	if err != nil {
		slog.Info("no psa_contact field found in zendesk - creating")
		uf, err = c.ZendeskClient.PostUserField(ctx, "integer", psaContactFieldKey, psaContactFieldTitle, psaFieldDescription)
		if err != nil {
			slog.Error("creating psa contact field", "error", err)
			return fmt.Errorf("creating psa contact field: %w", err)
		}
	}

	cf, err := c.ZendeskClient.GetOrgFieldByKey(ctx, psaCompanyFieldKey)
	if err != nil {
		slog.Info("no psa_company field found in zendesk - creating")
		cf, err = c.ZendeskClient.PostOrgField(ctx, "integer", psaCompanyFieldKey, psaCompanyFieldTitle, psaFieldDescription)
		if err != nil {
			slog.Error("creating psa company field", "error", err)
			return fmt.Errorf("creating psa company field: %w", err)
		}
	}

	c.Cfg.Zendesk.FieldIds.PsaContactId = uf.Id
	c.Cfg.Zendesk.FieldIds.PsaCompanyId = cf.Id
	viper.Set("zendesk.field_ids.psa_contact_id", uf.Id)
	viper.Set("zendesk.field_ids.psa_company_id", cf.Id)
	if err := viper.WriteConfig(); err != nil {
		slog.Error("writing zendesk psa fields to config", "error", err)
		return fmt.Errorf("writing zendesk psa fields to config: %w", err)
	}
	slog.Debug("CheckZendeskPSAFields", "userField", c.Cfg.Zendesk.FieldIds.PsaContactId, "orgField", c.Cfg.Zendesk.FieldIds.PsaCompanyId)
	return nil
}

func (c *Client) runBoardForm(ctx context.Context) error {
	boards, err := c.CwClient.GetBoards(ctx)
	if err != nil {
		return fmt.Errorf("getting boards: %w", err)
	}

	var boardNames []string
	boardsMap := make(map[string]int)

	for _, board := range boards {
		boardNames = append(boardNames, board.Name)
		boardsMap[board.Name] = board.Id
	}

	sort.Strings(boardNames)
	var s string
	input := huh.NewSelect[string]().
		Title("Choose destination ConnectWise PSA board").
		Options(huh.NewOptions(boardNames...)...).
		Value(&s).
		WithTheme(huh.ThemeBase16())

	if err := input.Run(); err != nil {
		return fmt.Errorf("running board form: %w", err)
	}

	if _, ok := boardsMap[s]; !ok {
		return errors.New("invalid board selection")
	}

	c.Cfg.CW.DestinationBoardId = boardsMap[s]
	viper.Set("connectwise_psa.destination_board_id", boardsMap[s])
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (c *Client) runBoardStatusForm(ctx context.Context, boardId int) error {
	statuses, err := c.CwClient.GetBoardStatuses(ctx, boardId)
	if err != nil {
		return fmt.Errorf("getting board statuses: %w", err)
	}

	var statusNames []string
	statusMap := make(map[string]int)
	for _, status := range statuses {
		statusNames = append(statusNames, status.Name)
		statusMap[status.Name] = status.Id
	}

	sort.Strings(statusNames)
	var cl, op string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose the Open status for the chosen board").
				Options(huh.NewOptions(statusNames...)...).
				Value(&op)),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose the Closed status for the chosen board").
				Options(huh.NewOptions(statusNames...)...).
				Value(&cl)),
	).WithShowHelp(false).WithTheme(huh.ThemeBase16())

	if err := form.Run(); err != nil {
		return fmt.Errorf("running board status form: %w", err)
	}

	if _, ok := statusMap[op]; !ok {
		return errors.New("invalid open status selection")
	}

	if _, ok := statusMap[cl]; !ok {
		return errors.New("invalid closed status selection")
	}

	c.Cfg.CW.OpenStatusId = statusMap[op]
	c.Cfg.CW.OpenStatusId = statusMap[cl]
	viper.Set("connectwise_psa.open_status_id", statusMap[op])
	viper.Set("connectwise_psa.closed_status_id", statusMap[cl])

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
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
