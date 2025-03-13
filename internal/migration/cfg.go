package migration

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/zendesk"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"time"
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
	Creds           zendesk.Creds `mapstructure:"api_creds" json:"api_creds"`
	TagsToMigrate   []TagDetails  `mapstructure:"tags_to_migrate" json:"tags_to_migrate"`
	FieldIds        ZdFieldIds    `mapstructure:"field_ids" json:"field_ids"`
	MasterStartDate string        `mapstructure:"start_date" json:"start_date"`
	MasterEndDate   string        `mapstructure:"end_date" json:"end_date"`
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
			slog.Info("creating default config file")
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

	if err := cfg.validateZendeskDates(); err != nil {
		if err := cfg.runZendeskDateForm(); err != nil {
			return fmt.Errorf("error running zendesk dates form: %w", err)
		}
	}

	if err := cfg.validateZendeskTags(); err != nil {
		if err := cfg.runZendeskTagsForm(); err != nil {
			return fmt.Errorf("error running zendesk tags form: %w", err)
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

	if err := cfg.runZendeskDateForm(); err != nil {
		slog.Error("error running date form", "error", err)
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
		huh.NewGroup(
			huh.NewInput().
				Title("Zendesk Token").
				Placeholder(cfg.Zendesk.Creds.Token).
				Validate(requiredInput).
				Inline(true).
				Value(&cfg.Zendesk.Creds.Token),
			huh.NewInput().
				Title("Zendesk Username").
				Placeholder(cfg.Zendesk.Creds.Username).
				Validate(requiredInput).
				Inline(true).
				Value(&cfg.Zendesk.Creds.Username),
			huh.NewInput().
				Title("Zendesk Subdomain").
				Placeholder(cfg.Zendesk.Creds.Subdomain).
				Validate(requiredInput).
				Inline(true).
				Value(&cfg.Zendesk.Creds.Subdomain),
			huh.NewInput().
				Title("ConnectWise Company ID").
				Placeholder(cfg.CW.Creds.CompanyId).
				Validate(requiredInput).
				Inline(true).
				Value(&cfg.CW.Creds.CompanyId),
			huh.NewInput().
				Title("ConnectWise Public Key").
				Placeholder(cfg.CW.Creds.PublicKey).
				Validate(requiredInput).
				Inline(true).
				Value(&cfg.CW.Creds.PublicKey),
			huh.NewInput().
				Title("ConnectWise Private Key").
				Placeholder(cfg.CW.Creds.PrivateKey).
				Validate(requiredInput).
				Inline(true).
				Value(&cfg.CW.Creds.PrivateKey),
			huh.NewInput().
				Title("ConnectWise Client ID").
				Placeholder(cfg.CW.Creds.ClientId).
				Validate(requiredInput).
				Inline(true).
				Value(&cfg.CW.Creds.ClientId),
		),
	).WithShowHelp(false).WithTheme(huh.ThemeBase16())
}

func (cfg *Config) validateZendeskDates() error {
	if err := validDateString(cfg.Zendesk.MasterStartDate); err != nil {
		// Set value in config to empty so the bad value isn't shown in the form
		cfg.Zendesk.MasterStartDate = ""
		slog.Warn("invalid zendesk start date string")
		return errors.New("invalid zendesk start date string")
	}

	if err := validDateString(cfg.Zendesk.MasterEndDate); err != nil {
		cfg.Zendesk.MasterEndDate = ""
		slog.Warn("invalid zendesk end date string")
		return errors.New("invalid zendesk end date string")
	}

	return nil
}

func (cfg *Config) runZendeskDateForm() error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Begin date to look for Zendesk tickets").
				Description("Use format YYYY-DD-MM (leave blank for no cutoff)").
				Placeholder(cfg.Zendesk.MasterStartDate).
				Validate(validDateString).
				Value(&cfg.Zendesk.MasterStartDate),
			huh.NewInput().
				Title("End date to look for Zendesk tickets").
				Description("Use format YYYY-DD-MM (leave blank for no cutoff)").
				Placeholder(cfg.Zendesk.MasterEndDate).
				Validate(validDateString).
				Value(&cfg.Zendesk.MasterEndDate),
		),
	).WithShowHelp(false).WithTheme(huh.ThemeBase16())

	if err := form.Run(); err != nil {
		return fmt.Errorf("error running date form: %w", err)
	}

	viper.Set("zendesk.start_date", cfg.Zendesk.MasterStartDate)
	viper.Set("zendesk.end_date", cfg.Zendesk.MasterEndDate)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
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

func (cfg *Config) runConnectwiseFieldForm() error {
	s := strconv.Itoa(cfg.CW.FieldIds.ZendeskTicketId)
	input := huh.NewInput().
		Title("Enter ConnectWise PSA custom field ID").
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

// Validator for required huh Input fields
func validDateString(s string) error {
	date, err := ConvertStringToTime(s)
	if err != nil {
		slog.Warn("error converting date string", "error", err)
		return errors.New("not a valid date string")
	}

	slog.Debug("valid date string", "date", date)
	return nil
}

func setCfgDefaults() {
	slog.Debug("setting config defaults")
	viper.SetDefault("zendesk", ZdCfg{})
	viper.SetDefault("connectwise_psa", CwCfg{})
}

func ConvertStringToTime(date string) (time.Time, error) {
	layout := "2006-01-02"
	d, err := time.Parse(layout, date)
	if err != nil {
		return time.Time{}, fmt.Errorf("converting time string to datetime format: %w", err)
	}

	return d, nil
}
