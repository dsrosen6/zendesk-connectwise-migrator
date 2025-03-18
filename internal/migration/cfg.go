package migration

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"github.com/spf13/viper"
	"log/slog"
	"sort"
	"time"
)

const (
	configFileSubPath = "/migrator_config.json"
)

var (
	CfgFile string
)

type Config struct {
	Zendesk     ZendeskConfig     `mapstructure:"zendesk" json:"zendesk"`
	Connectwise ConnectwiseConfig `mapstructure:"connectwise_psa" json:"connectwise_psa"`
}

type ZendeskConfig struct {
	Creds           zendesk.Creds   `mapstructure:"api_creds" json:"api_creds"`
	TagsToMigrate   []TagDetails    `mapstructure:"tags_to_migrate" json:"tags_to_migrate"`
	FieldIds        ZendeskFieldIds `mapstructure:"field_ids" json:"field_ids"`
	MasterStartDate string          `mapstructure:"start_date" json:"start_date"`
	MasterEndDate   string          `mapstructure:"end_date" json:"end_date"`

	// temporary values to assist with config forms - they do not get written to the config file.
	tempTagNames    []string
	tempTagsString  string
	wantTagDateForm bool
}

type ConnectwiseConfig struct {
	Creds              psa.Creds           `mapstructure:"api_creds" json:"api_creds"`
	ClosedStatusId     int                 `mapstructure:"closed_status_id" json:"closed_status_id"`
	OpenStatusId       int                 `mapstructure:"open_status_id" json:"open_status_id"`
	DestinationBoardId int                 `mapstructure:"destination_board_id" json:"destination_board_id"`
	FieldIds           ConnectwiseFieldIds `mapstructure:"field_ids" json:"field_ids"`

	// temporary values to assist with config forms - they do not get written to the config file.
	tempCwTagString string
}

type ZendeskFieldIds struct {
	PsaCompanyId int64 `mapstructure:"psa_company_id" json:"psa_company_id"`
	PsaContactId int64 `mapstructure:"psa_contact_id" json:"psa_contact_id"`
}

type ConnectwiseFieldIds struct {
	ZendeskTicketId int `mapstructure:"zendesk_ticket_id" json:"zendesk_ticket_id"`
}

// InitConfig reads in config file
func InitConfig(dir string) (*Config, error) {
	// Find home directory.
	configDir := dir

	if CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(CfgFile)
	} else {
		// Search config in home directory with name "migrator_config" (without extension).
		viper.AddConfigPath(configDir)
		viper.SetConfigType("json")
		viper.SetConfigName("migrator_config")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			setCfgDefaults()
			path := configDir + configFileSubPath
			slog.Info("creating default config file")
			if err := viper.WriteConfigAs(path); err != nil {
				return nil, fmt.Errorf("creating default config file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return cfg, nil
}

// validatePreClient validates config variables that do not require API connections
func (cfg *Config) validatePreClient() error {
	valid := true
	if err := cfg.validateCreds(); err != nil {
		slog.Error("missing required config values", "error", err)
		valid = false
	}

	if err := cfg.validateZendeskDates(); err != nil {
		slog.Error("zendesk master dates are invalid", "error", err)
		valid = false
	}

	if len(cfg.Zendesk.TagsToMigrate) == 0 {
		slog.Warn("no zendesk tags to migrate set")
		valid = false

		fmt.Println("\nNo Zendesk tags to migrate set - please enter comma-separated tags in the config file")
	}

	if cfg.Connectwise.FieldIds.ZendeskTicketId == 0 {
		slog.Warn("no ConnectWise PSA custom field ID set")
		valid = false

		fmt.Println("\nNo ConnectWise PSA custom field ID set - please enter the ID in the config file")
	}

	if !valid {
		return errors.New("one or more config values are invalid")
	}

	return nil
}

// validatePostClient runs after the client has been created, since we need valid API connections
// to validate these fields
func (c *Client) validatePostClient(ctx context.Context) error {
	action := func(ctx context.Context) error {
		return c.testConnection(ctx)
	}

	if err := runSpinner("Testing API connections", action); err != nil {
		return fmt.Errorf("testing API connections: %w", err)
	}

	if err := c.Cfg.validateZendeskCustomFields(); err != nil {
		proceed, err := confirmProcessZendeskFields()
		if err != nil {
			return err
		}

		if !proceed {
			return errors.New("custom field creation cancelled")
		}

		action := func(ctx context.Context) error {
			return c.processZendeskPsaFields(ctx)
		}
		if err := runSpinner("Processing Zendesk custom fields", action); err != nil {
			return fmt.Errorf("creating Zendesk custom fields: %w", err)
		}
	}

	if err := c.Cfg.validateConnectwiseBoardId(); err != nil {
		if err := c.runBoardForm(ctx); err != nil {
			return fmt.Errorf("running board form: %w", err)
		}
	}

	if err := c.Cfg.validateConnectwiseStatuses(); err != nil {
		if err := c.runBoardStatusForm(ctx, c.Cfg.Connectwise.DestinationBoardId); err != nil {
			return fmt.Errorf("running board status form: %w", err)
		}
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
		"connectwise_psa.api_creds.company_id":  cfg.Connectwise.Creds.CompanyId,
		"connectwise_psa.api_creds.public_key":  cfg.Connectwise.Creds.PublicKey,
		"connectwise_psa.api_creds.private_key": cfg.Connectwise.Creds.PrivateKey,
		"connectwise_psa.api_creds.client_id":   cfg.Connectwise.Creds.ClientId,
	}

	for k, v := range requiredFields {
		if v == "" {
			slog.Warn("missing required config value", "key", k)
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		slog.Error("missing required config values", "missing", missing)

		fmt.Println("\nThe following required API credential fields are missing:")
		for _, v := range missing {
			fmt.Printf("  - %s\n", v)
		}

		return errors.New("missing 1 or more required config values")
	}

	slog.Debug("all required api credentials fields found")
	return nil
}

func (cfg *Config) validateZendeskDates() error {
	valid := true
	var invalidDates []string
	if err := validDateString(cfg.Zendesk.MasterStartDate); err != nil {
		invalidDates = append(invalidDates, "master start date")
		slog.Warn("invalid master zendesk start date string")
		valid = false
	}

	if err := validDateString(cfg.Zendesk.MasterEndDate); err != nil {
		invalidDates = append(invalidDates, "master end date")
		slog.Warn("invalid master zendesk end date string")
		valid = false
	}

	for _, tag := range cfg.Zendesk.TagsToMigrate {
		if err := validDateString(tag.StartDate); err != nil {
			invalidDates = append(invalidDates, fmt.Sprintf("tag %s start date", tag.Name))
			valid = false
		}

		if err := validDateString(tag.EndDate); err != nil {
			invalidDates = append(invalidDates, fmt.Sprintf("tag %s end date", tag.Name))
			valid = false
		}
	}

	if !valid {
		slog.Error("invalid zendesk cutoff dates", "invalidDates", invalidDates)
		fmt.Println("\nThe following Zendesk cutoff date fields are invalid\n" +
			"Please ensure they are in the format 'YYYY-MM-DD'")

		for i, v := range invalidDates {
			fmt.Printf("  - %s\n", v)

			if i == len(invalidDates)-1 {
				// I have diagnosed OCD
				fmt.Println()
			}
		}

		return errors.New("invalid zendesk cutoff dates")
	}

	slog.Debug("all zendesk dates are valid")
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

func confirmProcessZendeskFields() (bool, error) {
	proceed := true
	input := huh.NewConfirm().
		Description("The migrator will check for the presence of the following custom fields in Zendesk:\n\n" +
			"- \"psa_company_id\" (integer Organization field)\n" +
			"- \"psa_contact_id\" (integer User field)\n\n" +
			"If these fields are not present, the migrator will create them for you.",
		).
		Value(&proceed).
		Affirmative("OK").
		Negative("Cancel")

	if err := input.WithKeyMap(customKeyMap()).WithTheme(CustomHuhTheme()).Run(); err != nil {
		return false, fmt.Errorf("running custom field confirmation input: %w", err)
	}
	return proceed, nil
}

func (c *Client) processZendeskPsaFields(ctx context.Context) error {
	uf, err := c.ZendeskClient.GetUserFieldByKey(ctx, psaContactFieldKey)
	if err != nil {
		slog.Info("no psa_contact field found in zendesk - creating")
		uf, err = c.ZendeskClient.PostUserField(ctx, "integer", psaContactFieldKey, psaContactFieldTitle, psaFieldDescription)
		if err != nil {
			return fmt.Errorf("creating psa contact field: %w", err)
		}
	}

	cf, err := c.ZendeskClient.GetOrgFieldByKey(ctx, psaCompanyFieldKey)
	if err != nil {
		slog.Info("no psa_company field found in zendesk - creating")
		cf, err = c.ZendeskClient.PostOrgField(ctx, "integer", psaCompanyFieldKey, psaCompanyFieldTitle, psaFieldDescription)
		if err != nil {
			return fmt.Errorf("creating psa company field: %w", err)
		}
	}

	c.Cfg.Zendesk.FieldIds.PsaContactId = uf.Id
	c.Cfg.Zendesk.FieldIds.PsaCompanyId = cf.Id
	viper.Set("zendesk.field_ids.psa_contact_id", uf.Id)
	viper.Set("zendesk.field_ids.psa_company_id", cf.Id)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

func (cfg *Config) validateConnectwiseBoardId() error {
	if cfg.Connectwise.DestinationBoardId == 0 {
		slog.Warn("no destination board ID set")
		return errors.New("no destination board ID set")
	}

	slog.Debug("connectwise board id found in config", "boardId", cfg.Connectwise.DestinationBoardId)
	return nil
}

func (c *Client) runBoardForm(ctx context.Context) error {
	var boards []psa.Board
	var err error
	action := func(ctx context.Context) error {
		boards, err = c.CwClient.GetBoards(ctx)
		return err
	}

	if err := runSpinner("Getting ConnectWise PSA boards", action); err != nil {
		return fmt.Errorf("getting ConnectWise PSA boards: %w", err)
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
		Value(&s)

	if err := input.WithKeyMap(customKeyMap()).WithTheme(CustomHuhTheme()).Run(); err != nil {
		return fmt.Errorf("running board form: %w", err)
	}

	if _, ok := boardsMap[s]; !ok {
		return errors.New("invalid board selection")
	}

	c.Cfg.Connectwise.DestinationBoardId = boardsMap[s]

	viper.Set("connectwise_psa.destination_board_id", boardsMap[s])
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (cfg *Config) validateConnectwiseStatuses() error {
	if cfg.Connectwise.OpenStatusId == 0 || cfg.Connectwise.ClosedStatusId == 0 {
		slog.Warn("no open status ID or closed status ID set")
		return errors.New("no open status ID or closed status ID set")
	}

	slog.Debug("board status ids", "open", cfg.Connectwise.OpenStatusId, "closed", cfg.Connectwise.ClosedStatusId)
	return nil
}

func (c *Client) runBoardStatusForm(ctx context.Context, boardId int) error {
	var statuses []psa.Status
	var err error
	action := func(ctx context.Context) error {
		statuses, err = c.CwClient.GetBoardStatuses(ctx, boardId)
		return err
	}

	if err := runSpinner("Getting ConnectWise PSA board statuses", action); err != nil {
		return fmt.Errorf("getting ConnectWise PSA board statuses: %w", err)
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
	).WithShowHelp(false).WithKeyMap(customKeyMap()).WithTheme(CustomHuhTheme())

	if err := form.Run(); err != nil {
		return fmt.Errorf("running board status form: %w", err)
	}

	if _, ok := statusMap[op]; !ok {
		return errors.New("invalid open status selection")
	}

	if _, ok := statusMap[cl]; !ok {
		return errors.New("invalid closed status selection")
	}

	c.Cfg.Connectwise.OpenStatusId = statusMap[op]
	c.Cfg.Connectwise.OpenStatusId = statusMap[cl]

	viper.Set("connectwise_psa.open_status_id", statusMap[op])
	viper.Set("connectwise_psa.closed_status_id", statusMap[cl])
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// Validator for required huh Input fields
func validDateString(s string) error {
	if s == "" {
		// blank is okay, it means no cutoff
		return nil
	}

	_, err := ConvertStringToTime(s)
	if err != nil {
		slog.Warn("error converting date string", "error", err)
		return errors.New("not a valid date string")
	}

	return nil
}

func setCfgDefaults() {
	slog.Debug("setting config defaults")
	viper.SetDefault("zendesk", ZendeskConfig{})
	viper.SetDefault("connectwise_psa", ConnectwiseConfig{})
}

func ConvertStringToTime(date string) (time.Time, error) {
	layout := "2006-01-02"
	d, err := time.Parse(layout, date)
	if err != nil {
		return time.Time{}, fmt.Errorf("converting time string to datetime format: %w", err)
	}

	return d, nil
}

func runSpinner(title string, action func(context.Context) error) error {
	return spinner.New().
		Title(fmt.Sprintf(" %s", title)).
		Type(spinner.Line).
		ActionWithErr(action).
		Style(lipgloss.NewStyle()).
		Run()
}
