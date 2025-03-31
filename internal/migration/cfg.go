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
	"os"
	"sort"
	"strconv"
)

const (
	configFileSubPath = "/config.json"
)

var (
	CfgFile string
)

type Config struct {
	TimeZone      string                  `mapstructure:"time_zone" json:"time_zone"` // for timestamps in tickets, ie "America/Chicago" - if not entered in config, defaults to UTC
	Zendesk       ZendeskConfig           `mapstructure:"zendesk" json:"zendesk"`
	Connectwise   ConnectwiseConfig       `mapstructure:"connectwise" json:"connectwise"`
	AgentMappings map[string]AgentMapping `mapstructure:"agent_mappings" json:"agent_mappings"`

	CliOptions
}

type CliOptions struct {
	Debug              bool
	TicketLimit        int          `mapstructure:"ticket_limit" json:"ticket_limit"`
	MigrateOpenTickets bool         `mapstructure:"migrate_open_tickets" json:"migrate_open_tickets"`
	OutputLevels       OutputLevels `mapstructure:"output_levels" json:"output_levels"`
	StopAfterOrgs      bool
	StopAfterUsers     bool
}

type OutputLevels struct {
	NoAction bool `mapstructure:"no_action" json:"no_action"`
	Created  bool `mapstructure:"created" json:"created"`
	Warn     bool `mapstructure:"warn" json:"warn"`
	Error    bool `mapstructure:"error" json:"error"`
}

type ZendeskConfig struct {
	Creds           zendesk.Creds   `mapstructure:"api_creds" json:"api_creds"`
	TagsToMigrate   []TagDetails    `mapstructure:"tags_to_migrate" json:"tags_to_migrate"`
	FieldIds        ZendeskFieldIds `mapstructure:"field_ids" json:"field_ids"`
	MasterStartDate string          `mapstructure:"start_date" json:"start_date"`
	MasterEndDate   string          `mapstructure:"end_date" json:"end_date"`
}

type TagDetails struct {
	Name      string `mapstructure:"name" json:"name"`
	StartDate string `mapstructure:"start_date" json:"start_date"`
	EndDate   string `mapstructure:"end_date" json:"end_date"`
}

type ConnectwiseConfig struct {
	Creds              psa.Creds           `mapstructure:"api_creds" json:"api_creds"`
	ClosedStatusId     int                 `mapstructure:"closed_status_id" json:"closed_status_id"`
	OpenStatusId       int                 `mapstructure:"open_status_id" json:"open_status_id"`
	TicketType         int                 `mapstructure:"ticket_type" json:"ticket_type"`
	DestinationBoardId int                 `mapstructure:"destination_board_id" json:"destination_board_id"`
	FieldIds           ConnectwiseFieldIds `mapstructure:"field_ids" json:"field_ids"`
}

type ZendeskFieldIds struct {
	PsaCompanyId int64 `mapstructure:"psa_company_id" json:"psa_company_id"`
	PsaContactId int64 `mapstructure:"psa_contact_id" json:"psa_contact_id"`
}

type ConnectwiseFieldIds struct {
	ZendeskTicketId   int `mapstructure:"zendesk_ticket_id" json:"zendesk_ticket_id"`
	ZendeskClosedDate int `mapstructure:"zendesk_closed_date" json:"zendesk_closed_date"`
}

type AgentMapping struct {
	Email string `mapstructure:"email_address" json:"email_address"`
	PsaId int    `mapstructure:"psa_member_id" json:"psa_member_id"`
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
		viper.SetConfigName("config")
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
			fmt.Printf("Default config file created at:\n%s\n\nPlease fill out the fields and then run the migration command again\n", path)
			os.Exit(0)
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

		fmt.Println("\nNo Zendesk tags to migrate set in config - enter at least one")
	} else {
		for _, tag := range cfg.Zendesk.TagsToMigrate {
			if tag.Name == "" {
				slog.Warn("tag name is empty")
				valid = false
				fmt.Println("\nOne or more tag names are empty - please enter a name for each tag")
				break
			} else if tag.Name == exampleTag1.Name || tag.Name == exampleTag2.Name {
				slog.Warn("example tags are present in config")
				valid = false
				fmt.Println("\nExample tags are present in config - replace them with your own tags")
				break
			}
		}
	}

	if cfg.Connectwise.FieldIds.ZendeskTicketId == 0 {
		slog.Warn("no ConnectWise PSA custom field ID set for Zendesk Ticket Number")
		valid = false

		fmt.Println("\nNo ConnectWise PSA custom field ID set for Zendesk Ticket Number in config")
	}

	if cfg.Connectwise.FieldIds.ZendeskClosedDate == 0 {
		slog.Warn("no ConnectWise PSA custom field ID set for Zendesk Closed Date in config")
		valid = false

		fmt.Println("\nNo ConnectWise PSA custom field ID set for Zendesk Closed Date in config")

	}

	if !valid {
		return errors.New("one or more config values are invalid - see above")
	}

	return nil
}

// validatePostClient runs after the Client has been created, since we need valid API connections
// to validate these fields
func (c *Client) validatePostClient(ctx context.Context) error {
	action := func(ctx context.Context) error {
		return c.testConnection(ctx)
	}

	if err := runConfigSpinner("Testing API connections", action); err != nil {
		return fmt.Errorf("testing API connections: %w", err)
	}

	action = func(ctx context.Context) error { return c.processAgentMappings(ctx) }
	if err := runConfigSpinner("Checking agent mappings", action); err != nil {
		return fmt.Errorf("processing agent mappings: %w", err)
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
		if err := runConfigSpinner("Processing Zendesk custom fields", action); err != nil {
			return fmt.Errorf("creating Zendesk custom fields: %w", err)
		}
	}

	if err := c.Cfg.validateConnectwiseBoardId(); err != nil {
		if err := c.runBoardForm(ctx); err != nil {
			return fmt.Errorf("running board form: %w", err)
		}
	}

	if err := c.Cfg.validateConnectwiseBoardType(); err != nil {
		if err := c.runTicketTypeForm(ctx, c.Cfg.Connectwise.DestinationBoardId); err != nil {
			return fmt.Errorf("running ticket type form: %w", err)
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
		"Zendesk API Token":           cfg.Zendesk.Creds.Token,
		"Zendesk API Username":        cfg.Zendesk.Creds.Username,
		"Zendesk API Subdomain":       cfg.Zendesk.Creds.Subdomain,
		"ConnectWise API Company ID":  cfg.Connectwise.Creds.CompanyId,
		"ConnectWise API Public Key":  cfg.Connectwise.Creds.PublicKey,
		"ConnectWise API Private Key": cfg.Connectwise.Creds.PrivateKey,
		"ConnectWise API client ID":   cfg.Connectwise.Creds.ClientId,
	}

	for k, v := range requiredFields {
		if v == "" {
			slog.Warn("missing required config value", "key", k)
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		slog.Error("missing required config values", "missing", missing)

		fmt.Println("\nThe following required API credential fields are missing in the config:")
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

func (c *Client) processAgentMappings(ctx context.Context) error {
	slog.Debug("processing agent mappings")
	if c.Cfg.AgentMappings == nil {
		c.Cfg.AgentMappings = make(map[string]AgentMapping)
	}

	zendeskAgents, err := c.ZendeskClient.GetAgents(ctx)
	if err != nil {
		return fmt.Errorf("getting zendesk agents: %w", err)
	}

	psaMembers, err := c.CwClient.GetMembers(ctx)
	if err != nil {
		return fmt.Errorf("getting connectwise psa members: %w", err)
	}

	for _, agent := range zendeskAgents {
		idString := strconv.Itoa(agent.Id)
		slog.Debug("checking agent mapping", "agentEmail", agent.Email, "zendeskId", agent.Id)
		var psaId int
		for _, member := range psaMembers {
			if agent.Email == member.PrimaryEmail {
				psaId = member.Id
				continue
			}
		}

		if psaId == 0 {
			slog.Warn("no matching psa member found for zendesk agent", "agent", agent.Email)
			continue
		}

		if _, ok := c.Cfg.AgentMappings[idString]; ok && c.Cfg.AgentMappings[idString].PsaId == psaId {
			slog.Debug("agent mapping already exists", "agentId", agent.Id, "psaId", psaId)
			continue
		}

		c.Cfg.AgentMappings[idString] = AgentMapping{Email: agent.Email, PsaId: psaId}
		slog.Info("created or updated agent mapping",
			"agentEmail", agent.Email,
			"zendeskId", agent.Id,
			"psaId", psaId)
	}

	viper.Set("agent_mappings", c.Cfg.AgentMappings)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing agent mappings to config file: %w", err)
	}

	return nil
}

func confirmProcessZendeskFields() (bool, error) {
	var proceed bool
	input := huh.NewConfirm().
		Description("The migrator will check for the presence of the following custom fields in Zendesk:\n\n" +
			"- \"psa_company_id\" (integer Organization field)\n" +
			"- \"psa_contact_id\" (integer User field)\n\n" +
			"If these fields are not present, the migrator will create them for you.",
		).
		Value(&proceed).
		Affirmative("OK").
		Negative("Cancel")

	if err := input.WithKeyMap(customKeyMap()).WithTheme(customFormTheme()).Run(); err != nil {
		return false, fmt.Errorf("running custom field confirmation input: %w", err)
	}
	return proceed, nil
}

func (c *Client) processZendeskPsaFields(ctx context.Context) error {
	uf, err := c.ZendeskClient.GetUserFieldByKey(ctx, psaContactFieldKey)
	if err != nil {
		slog.Debug("no psa_contact field found in zendesk - creating")
		uf, err = c.ZendeskClient.PostUserField(ctx, "integer", psaContactFieldKey, psaContactFieldTitle, psaFieldDescription)
		if err != nil {
			return fmt.Errorf("creating psa contact field: %w", err)
		}
	}

	cf, err := c.ZendeskClient.GetOrgFieldByKey(ctx, psaCompanyFieldKey)
	if err != nil {
		slog.Debug("no psa_company field found in zendesk - creating")
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
	action := func(_ context.Context) error {
		boards, err = c.CwClient.GetBoards(ctx)
		return err
	}

	if err := runConfigSpinner("Getting ConnectWise PSA boards", action); err != nil {
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

	if err := input.WithKeyMap(customKeyMap()).WithTheme(customFormTheme()).Run(); err != nil {
		return fmt.Errorf("running board form: %w", err)
	}

	if _, ok := boardsMap[s]; !ok {
		return errors.New("invalid board selection")
	}

	c.Cfg.Connectwise.DestinationBoardId = boardsMap[s]

	viper.Set("connectwise.destination_board_id", boardsMap[s])
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (cfg *Config) validateConnectwiseBoardType() error {
	if cfg.Connectwise.TicketType == 0 {
		slog.Warn("no ticket type set")
		return errors.New("no ticket type set")
	}

	slog.Debug("ticket type id", "id", cfg.Connectwise.TicketType)
	return nil
}

func (c *Client) runTicketTypeForm(ctx context.Context, boardId int) error {
	var statuses []psa.BoardType
	var err error
	action := func(_ context.Context) error {
		statuses, err = c.CwClient.GetBoardTypes(ctx, boardId)
		return err
	}

	if err := runConfigSpinner("Getting ConnectWise PSA ticket types", action); err != nil {
		return fmt.Errorf("getting ConnectWise PSA ticket types: %w", err)
	}

	var types []string
	typeMap := make(map[string]int)
	for _, t := range statuses {
		types = append(types, t.Name)
		typeMap[t.Name] = t.Id
	}

	sort.Strings(types)
	var tp string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose the type you'd like tickets created as").
				Options(huh.NewOptions(types...)...).
				Value(&tp)),
	).WithShowHelp(false).WithKeyMap(customKeyMap()).WithTheme(customFormTheme())

	if err := form.Run(); err != nil {
		return fmt.Errorf("running ticket type form: %w", err)
	}

	if _, ok := typeMap[tp]; !ok {
		return errors.New("invalid type selection")
	}

	c.Cfg.Connectwise.TicketType = typeMap[tp]

	viper.Set("connectwise.ticket_type", typeMap[tp])
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
	action := func(_ context.Context) error {
		statuses, err = c.CwClient.GetBoardStatuses(ctx, boardId)
		return err
	}

	if err := runConfigSpinner("Getting ConnectWise PSA board statuses", action); err != nil {
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
	).WithShowHelp(false).WithKeyMap(customKeyMap()).WithTheme(customFormTheme())

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

	viper.Set("connectwise.open_status_id", statusMap[op])
	viper.Set("connectwise.closed_status_id", statusMap[cl])
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

	_, err := convertStrTime(s)
	if err != nil {
		slog.Warn("error converting date string", "error", err)
		return errors.New("not a valid date string")
	}

	return nil
}

var exampleTag1 = TagDetails{
	Name:      "example_tag_1",
	StartDate: "2021-01-01",
	EndDate:   "2021-12-31",
}

var exampleTag2 = TagDetails{
	Name:      "example_tag_2",
	StartDate: "2021-01-01",
	EndDate:   "",
}

var defaultOutputLevels = OutputLevels{
	NoAction: false,
	Created:  false,
	Warn:     true,
	Error:    true,
}

func setCfgDefaults() {
	slog.Debug("setting config defaults")
	viper.SetDefault("ticket_limit", 0)
	viper.SetDefault("migrate_open_tickets", false)
	viper.SetDefault("time_zone", "America/Chicago")
	viper.SetDefault("zendesk", ZendeskConfig{TagsToMigrate: []TagDetails{exampleTag1, exampleTag2}})
	viper.SetDefault("connectwise", ConnectwiseConfig{})
	viper.SetDefault("output_levels", defaultOutputLevels)
}

func runConfigSpinner(title string, action func(context.Context) error) error {
	return spinner.New().
		Title(fmt.Sprintf(" %s", title)).
		Type(spinner.Line).
		ActionWithErr(action).
		Style(lipgloss.NewStyle()).
		Run()
}
