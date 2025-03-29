package migration

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"strconv"
	"strings"
)

func (m *Model) getUsersToMigrate(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		slog.Debug("getUsersToMigrate: called", "orgName", org.ZendeskOrg.Name)
		users, err := m.client.ZendeskClient.GetOrganizationUsers(m.ctx, org.ZendeskOrg.Id)
		if err != nil {
			slog.Error("getting users for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("couldn't get users for org %s", org.ZendeskOrg.Name)), errOutput)
			m.orgsCheckedForUsers++
			m.userMigrationErrors++
			return nil
		}
		slog.Info("got users for org", "orgName", org.ZendeskOrg.Name, "totalUsers", len(users))

		for _, user := range users {
			idString := strconv.Itoa(user.Id)
			m.data.UsersToMigrate[idString] = &userMigrationDetails{ZendeskUser: &user, PsaCompany: org.PsaOrg}
		}

		m.orgsCheckedForUsers++
		return nil
	}
}

func (m *Model) migrateUser(user *userMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		if user.ZendeskUser.Email == "" {
			slog.Warn("zendesk user has no email address - skipping", "userName", user.ZendeskUser.Name)
			m.writeToOutput(warnYellowOutput("WARN", fmt.Sprintf("user has no email address, skipping migration: %s", user.ZendeskUser.Name)), warnOutput)
			m.usersProcessed++
			return nil
		}

		var err error
		user.PsaContact, err = m.matchZdUserToCwContact(user.ZendeskUser)
		if err != nil {

			if errors.Is(err, psa.NoUserFoundErr{}) {
				slog.Debug("user does not exist in psa - attempting to create new user", "userEmail", user.ZendeskUser.Email)
				user.PsaContact, err = m.createPsaContact(user)
				if err != nil {
					slog.Error("creating user", "userEmail", user.ZendeskUser.Email, "zendeskUserId", user.ZendeskUser.Id, "error", err)
					m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("creating user %s", user.ZendeskUser.Email)), errOutput)
					m.usersProcessed++
					m.userMigrationErrors++
					return nil
				} else {
					slog.Debug("created new psa user", "userName", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
				}

			} else {
				slog.Error("matching zendesk user to psa user", "userEmail", user.ZendeskUser.Email, "zendeskUserId", user.ZendeskUser.Id, "error", err)
				m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("matching zendesk user to psa user %s", user.ZendeskUser.Email)), errOutput)
				m.usersProcessed++
				m.userMigrationErrors++
				return nil
			}

		} else {
			slog.Debug("matched zendesk user to psa user", "userEmail", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
		}

		if user.ZendeskUser.UserFields.PSAContactId != user.PsaContact.Id {
			if err := m.updateContactFieldValue(user); err != nil {
				slog.Error("updating user contact field value in zendesk", "userEmail", user.ZendeskUser.Email, "zendeskUserId", user.ZendeskUser.Id, "psaContactId", user.PsaContact.Id, "error", err)
				m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("updating contact field in zendesk for %s", user.ZendeskUser.Email)), errOutput)
				m.usersProcessed++
				m.userMigrationErrors++
				return nil
			}

			slog.Info("user migrated", "userEmail", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
			m.writeToOutput(goodGreenOutput("SUCCESS", fmt.Sprintf("User fully migrated: %s", user.ZendeskUser.Email)), createdOutput)
			m.data.UsersInPsa[strconv.Itoa(user.ZendeskUser.Id)] = user
			m.usersProcessed++
			return nil

		} else {
			m.writeToOutput(goodBlueOutput("NO ACTION", fmt.Sprintf("user already existing in PSA: %s", user.ZendeskUser.Email)), noActionOutput)
			m.data.UsersInPsa[strconv.Itoa(user.ZendeskUser.Id)] = user
			m.usersProcessed++
			return nil
		}
	}
}

func (m *Model) matchZdUserToCwContact(user *zendesk.User) (*psa.Contact, error) {
	contact, err := m.client.CwClient.GetContactByEmail(m.ctx, user.Email)
	if err != nil {
		return nil, err
	}
	return contact, nil
}

func (m *Model) createPsaContact(user *userMigrationDetails) (*psa.Contact, error) {
	c := &psa.ContactPostBody{}
	c.FirstName, c.LastName = separateName(user.ZendeskUser.Name)

	if user.PsaCompany == nil {
		return nil, errors.New("user psa company is nil")
	}

	c.Company.Id = user.PsaCompany.Id

	c.CommunicationItems = []psa.CommunicationItem{
		{
			Type:              psa.CommunicationItemType{Name: "Email"},
			Value:             user.ZendeskUser.Email,
			CommunicationType: "Email",
		},
	}

	return m.client.CwClient.PostContact(m.ctx, c)
}

type ZendeskFieldAlreadySetErr struct{}

func (e ZendeskFieldAlreadySetErr) Error() string {
	return "zendesk user already has psa contact id field"
}

func (m *Model) updateContactFieldValue(user *userMigrationDetails) error {
	if user.ZendeskUser.UserFields.PSAContactId == user.PsaContact.Id {
		return ZendeskFieldAlreadySetErr{}
	}

	if user.PsaContact.Id != 0 {
		user.ZendeskUser.UserFields.PSAContactId = user.PsaContact.Id

		var err error
		user.ZendeskUser, err = m.client.ZendeskClient.UpdateUser(m.ctx, user.ZendeskUser)
		if err != nil {
			return fmt.Errorf("updating user with PSA contact id: %w", err)
		}

		slog.Debug("updated zendesk user with PSA contact id", "userEmail", user.ZendeskUser.Email)
		return nil
	} else {
		slog.Error("user psa id is 0 - cannot update psa_contact field in zendesk", "userEmail", user.ZendeskUser.Email, "zendeskUserId", user.ZendeskUser.Id)
		return errors.New("user psa id is 0 - cannot update psa_contact field in zendesk")
	}
}

func separateName(name string) (string, string) {
	nameParts := strings.Split(name, " ")
	if len(nameParts) == 1 {
		return nameParts[0], ""
	}

	return nameParts[0], strings.Join(nameParts[1:], " ")
}
