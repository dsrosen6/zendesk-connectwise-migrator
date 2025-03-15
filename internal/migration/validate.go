package migration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

func (cfg *Config) PreClientValidate() error {
	valid := true
	if err := cfg.validateCreds(); err != nil {
		slog.Warn("missing required config values", "error", err)
		valid = false
	}

	if err := cfg.validateZendeskDates(); err != nil {
		slog.Warn("zendesk master dates are invalid", "error", err)
	}

	if err := cfg.validateZendeskTags(); err != nil {
		slog.Warn("zendesk tags are invalid", "error", err)
	}

	if err := cfg.validateConnectwiseCustomField(); err != nil {
		slog.Warn("connectwise custom field id is invalid", "error", err)
	}

	if !valid {
		if err := cfg.RunForm(); err != nil {
			return fmt.Errorf("prompting fields: %w", err)
		}
	}

	return nil
}

func (c *Client) PostClientValidate(ctx context.Context) error {
	if err := c.testConnection(ctx); err != nil {
		slog.Error("ConnectionTest: error", "error", err)
		return err
	}

	if err := c.Cfg.validateZendeskCustomFields(); err != nil {
		if err := c.processZendeskPsaForms(ctx); err != nil {
			return fmt.Errorf("getting zendesk fields: %w", err)
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
		return errors.New("missing 1 or more required config values")
	}

	return nil
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

func (cfg *Config) validateZendeskTags() error {
	if len(cfg.Zendesk.TagsToMigrate) == 0 {
		slog.Warn("no zendesk tags to migrate set")
		return errors.New("no zendesk tags to migrate set")
	}

	for _, tag := range cfg.Zendesk.TagsToMigrate {
		if err := cfg.validateTagDates(&tag); err != nil {
			return fmt.Errorf("error validating tag dates: %w", err)
		}
	}

	return nil
}

func (cfg *Config) validateTagDates(tag *TagDetails) error {
	valid := true

	if err := validDateString(tag.StartDate); err != nil {
		tag.StartDate = ""
		slog.Warn("invalid zendesk start date string - replaced with blank string", "tag", tag.Name)
		valid = false
	}

	if err := validDateString(tag.EndDate); err != nil {
		tag.StartDate = ""
		slog.Warn("invalid zendesk end date string - replaced with blank string", "tag", tag.Name)
		valid = false
	}

	if !valid {
		return fmt.Errorf("invalid date(s) for tag %s", tag.Name)
	}

	return nil
}

func (cfg *Config) validateConnectwiseCustomField() error {
	if cfg.Connectwise.FieldIds.ZendeskTicketId == 0 {
		slog.Warn("no ConnectWise PSA custom field ID set")
		return errors.New("no ConnectWise PSA custom field ID set")
	}

	slog.Debug("connectwise custom field id found in config", "zendeskTicketId", cfg.Connectwise.FieldIds.ZendeskTicketId)
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
	if cfg.Connectwise.DestinationBoardId == 0 {
		slog.Warn("no destination board ID set")
		return errors.New("no destination board ID set")
	}

	slog.Debug("connectwise board id found in config", "boardId", cfg.Connectwise.DestinationBoardId)
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
