package zendesk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

type UsersResp struct {
	Users []User `json:"users"`
	Meta  Meta   `json:"meta"`
	Links Links  `json:"links"`
}

type UserBody struct {
	User *User `json:"user"`
}

type UserResp struct {
	User User `json:"user"`
}

type User struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	UserFields struct {
		PSAContactId int `json:"psa_contact"`
	} `json:"user_fields"`
}

func (c *Client) UpdateUser(ctx context.Context, user *User) (*User, error) {
	url := fmt.Sprintf("%s/users/%d", c.baseUrl, user.Id)

	b := &UserBody{
		User: user,
	}

	jsonBytes, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("marshaling user to json: %w", err)
	}

	body := bytes.NewReader(jsonBytes)

	u := &UserBody{User: user}
	if err := c.ApiRequest(ctx, "PUT", url, body, u); err != nil {
		return nil, fmt.Errorf("an error occured updating the user: %w", err)
	}

	return u.User, nil
}

func (c *Client) GetUser(ctx context.Context, userId int64) (*User, error) {
	url := fmt.Sprintf("%s/users/%d", c.baseUrl, userId)
	u := &UserResp{}

	if err := c.ApiRequest(ctx, "GET", url, nil, &u); err != nil {
		return nil, fmt.Errorf("an error occured getting the user: %w", err)
	}

	return &u.User, nil
}

func (c *Client) GetOrganizationUsers(ctx context.Context, orgId int64) ([]User, error) {
	initialUrl := fmt.Sprintf("%s/organizations/%d/users?page[size]=100", c.baseUrl, orgId)
	var allUsers []User
	currentPage := &UsersResp{}

	if err := c.ApiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting organization users: %w", err)
	}

	allUsers = append(allUsers, currentPage.Users...)

	for currentPage.Meta.HasMore {
		nextPage := &UsersResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting organization users: %w", err)
		}

		allUsers = append(allUsers, nextPage.Users...)
		currentPage = nextPage
	}

	return allUsers, nil
}

func (c *Client) GetAgents(ctx context.Context) ([]User, error) {
	initialUrl := fmt.Sprintf("%s/users?page[size]=100&role[]=admin&role[]=agent", c.baseUrl)
	var allAgents []User
	currentPage := &UsersResp{}

	if err := c.ApiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting agents: %w", err)
	}

	allAgents = append(allAgents, currentPage.Users...)

	for currentPage.Meta.HasMore {
		nextPage := &UsersResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting agents: %w", err)
		}

		allAgents = append(allAgents, nextPage.Users...)
		currentPage = nextPage
	}

	return allAgents, nil
}
