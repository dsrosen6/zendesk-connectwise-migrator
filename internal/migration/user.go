package migration

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"strings"
)

func (c *Client) MigrateUser(ctx context.Context, zendeskUser zendesk.User, psaCompanyId int) (*psa.Contact, error) {
	// Check if the Zendesk user already has a PSA contact id value - if so, they've already been migrated by this tool
	if zendeskUser.User.UserFields.PSAContactId != 0 {
		slog.Info("user already migrated",
			"zendeskUserId", zendeskUser.User.Id,
			"psaContactId", zendeskUser.User.UserFields.PSAContactId)
		return nil, nil
	}

	slog.Info("user migration started",
		"zendeskUserId", zendeskUser.User.Id,
		"psaCompanyId", psaCompanyId,
		"psaContactId", zendeskUser.User.UserFields.PSAContactId)

	contact, err := c.CwClient.GetContactByEmail(ctx, zendeskUser.User.Email)
	if err != nil {
		return nil, fmt.Errorf("getting contact by email: %w", err)
	}

	var alreadyExisted bool
	if contact != nil {
		alreadyExisted = true
	} else {
		firstName, lastName := separateName(zendeskUser.User.Name)
		contact = &psa.Contact{
			FirstName: firstName,
			LastName:  lastName,
			Company: psa.Company{
				Id: psaCompanyId,
			},
			CommunicationItems: []psa.CommunicationItem{
				{
					Type:  psa.CommunicationItemType{Name: "Email"},
					Value: zendeskUser.User.Email,
				},
			},
		}

		contact, err = c.CwClient.PostContact(ctx, contact)
		if err != nil {
			return nil, fmt.Errorf("creating new contact: %w", err)
		}

		slog.Debug("contact created in PSA", "contactId", contact.Id)
	}

	zendeskUser.User.UserFields.PSAContactId = contact.Id
	if err := c.ZendeskClient.UpdateUser(ctx, &zendeskUser); err != nil {
		return nil, fmt.Errorf("updating zendesk user with PSA contact id: %w", err)
	}

	slog.Info("finished migrating user to connectwise",
		"zendeskUserId", zendeskUser.User.Id,
		"psaContactId", contact.Id,
		"alreadyExisted", alreadyExisted)
	return contact, nil
}

func separateName(name string) (string, string) {
	nameParts := strings.Split(name, " ")
	if len(nameParts) == 1 {
		return nameParts[0], ""
	}

	return nameParts[0], strings.Join(nameParts[1:], " ")
}
