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
func InitConfig() (*Config, error) {
	// Find home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	if CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(CfgFile)
	} else {
		// Search config in home directory with name "migrator_config" (without extension).
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

func (cfg *Config) RunForm() error {
	if err := cfg.preProcessMainForm(); err != nil {
		return fmt.Errorf("error pre processing config form: %w", err)
	}

	if err := cfg.mainForm().Run(); err != nil {
		if errors.As(err, &huh.ErrUserAborted) {
			os.Exit(0)
		}
		return fmt.Errorf("running config form: %w", err)
	}

	if err := cfg.postProcessMainForm(); err != nil {
		return fmt.Errorf("post processing config form: %w", err)
	}

	if cfg.Zendesk.wantTagDateForm {
		if err := cfg.runZendeskTagDateForm(); err != nil {
			return fmt.Errorf("running tag date form: %w", err)
		}
	}

	viper.Set("zendesk", cfg.Zendesk)
	viper.Set("connectwise_psa", cfg.Connectwise)

	slog.Debug("writing config", "config", cfg)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing to config file: %w", err)
	}

	return nil
}

// preProcessMainForm converts various values from the config file to usable values for the config forms.
func (cfg *Config) preProcessMainForm() error {
	// Prepare for cfg.tagEntryGroup
	for _, tag := range cfg.Zendesk.TagsToMigrate {
		cfg.Zendesk.tempTagNames = append(cfg.Zendesk.tempTagNames, tag.Name)
	}
	cfg.Zendesk.tempTagsString = strings.Join(cfg.Zendesk.tempTagNames, ",")

	// Prepare for cfg.connectwiseCustomFieldGroup
	cfg.Connectwise.tempCwTagString = strconv.Itoa(cfg.Connectwise.FieldIds.ZendeskTicketId)

	// Set wantTagDateForm to true
	cfg.Zendesk.wantTagDateForm = true
	return nil
}

// postProcessMainForm converts the values output by the config forms back to the default values in the config file.
func (cfg *Config) postProcessMainForm() error {
	// Post process from cfg.tagEntryGroup
	var updatedTags []TagDetails
	cfg.Zendesk.tempTagNames = strings.Split(cfg.Zendesk.tempTagsString, ",")
	for _, tagName := range cfg.Zendesk.tempTagNames {
		if existingTag := findTagByName(cfg.Zendesk.TagsToMigrate, strings.TrimSpace(tagName)); existingTag != nil {
			slog.Debug("tag already exists", "tag", existingTag.Name)
			updatedTags = append(updatedTags, *existingTag)
		} else {
			slog.Debug("adding new tag", "tag", tagName)
			updatedTags = append(updatedTags, TagDetails{Name: strings.TrimSpace(tagName)})
		}
	}

	cfg.Zendesk.TagsToMigrate = updatedTags
	viper.Set("zendesk.tags_to_migrate", cfg.Zendesk.TagsToMigrate)

	var err error
	cfg.Connectwise.FieldIds.ZendeskTicketId, err = strToInt(cfg.Connectwise.tempCwTagString)
	if err != nil {
		return fmt.Errorf("converting connectwise zendesk ticket id field to int: %w", err)
	}

	viper.Set("connectwise_psa.field_ids.zendesk_ticket_id", cfg.Connectwise.FieldIds.ZendeskTicketId)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (cfg *Config) mainForm() *huh.Form {
	return huh.NewForm(
		cfg.credsGroup(),
		cfg.connectwiseCustomFieldGroup(),
		cfg.tagEntryGroup(),
		cfg.masterDateGroup(),
		cfg.tagDateWarningGroup(),
	).WithShowHelp(false).WithKeyMap(customKeyMap()).WithTheme(CustomHuhTheme())
}

func (cfg *Config) credsGroup() *huh.Group {
	return huh.NewGroup(
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
			Placeholder(cfg.Connectwise.Creds.CompanyId).
			Validate(requiredInput).
			Inline(true).
			Value(&cfg.Connectwise.Creds.CompanyId),
		huh.NewInput().
			Title("ConnectWise Public Key").
			Placeholder(cfg.Connectwise.Creds.PublicKey).
			Validate(requiredInput).
			Inline(true).
			Value(&cfg.Connectwise.Creds.PublicKey),
		huh.NewInput().
			Title("ConnectWise Private Key").
			Placeholder(cfg.Connectwise.Creds.PrivateKey).
			Validate(requiredInput).
			Inline(true).
			Value(&cfg.Connectwise.Creds.PrivateKey),
		huh.NewInput().
			Title("ConnectWise Client ID").
			Placeholder(cfg.Connectwise.Creds.ClientId).
			Validate(requiredInput).
			Inline(true).
			Value(&cfg.Connectwise.Creds.ClientId),
	)
}

func (cfg *Config) masterDateGroup() *huh.Group {
	return huh.NewGroup(
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
	)
}

func (cfg *Config) tagEntryGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Enter Zendesk tags to migrate").
			Placeholder(cfg.Zendesk.tempTagsString).
			Description("Separate tags by commas, and then press Enter").
			Validate(requiredInput).
			Value(&cfg.Zendesk.tempTagsString),
	)
}

func (cfg *Config) tagDateWarningGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[bool]().
			Title("Should your tags to have different cutoff dates than the master dates?").
			Options(
				huh.NewOption("Yes", true),
				huh.NewOption("No", false),
			).
			Value(&cfg.Zendesk.wantTagDateForm),
	)
}

func (cfg *Config) connectwiseCustomFieldGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Enter ConnectWise PSA custom field ID").
			Description("See docs if you have not made one.").
			Placeholder(cfg.Connectwise.tempCwTagString).
			Validate(requiredInput).
			Value(&cfg.Connectwise.tempCwTagString),
	)
}

func (cfg *Config) runConnectwiseFieldForm() error {
	s := strconv.Itoa(cfg.Connectwise.FieldIds.ZendeskTicketId)
	input := huh.NewInput().
		Title("Enter ConnectWise PSA custom field ID").
		Description("See docs if you have not made one.").
		Placeholder(s).
		Validate(requiredInput).
		Value(&s).
		WithTheme(CustomHuhTheme())

	if err := input.Run(); err != nil {
		return fmt.Errorf("running custom field form: %w", err)
	}

	i, err := strToInt(s)
	if err != nil {
		return err
	}

	viper.Set("connectwise_psa.field_ids.zendesk_ticket_id", i)

	return nil
}

func (c *Client) processZendeskPsaForms(ctx context.Context) error {
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
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose destination ConnectWise PSA board").
				Options(huh.NewOptions(boardNames...)...).
				Value(&s),
		),
	).WithShowHelp(false).WithKeyMap(customKeyMap()).WithTheme(CustomHuhTheme())

	if err := form.Run(); err != nil {
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

func strToInt(s string) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
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
