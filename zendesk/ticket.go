package zendesk

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Ticket struct {
	Ticket struct {
		Url        string      `json:"url"`
		Id         int         `json:"id"`
		ExternalId interface{} `json:"external_id"`
		Via        struct {
			Channel string `json:"channel"`
			Source  struct {
				From struct {
				} `json:"from"`
				To struct {
				} `json:"to"`
				Rel interface{} `json:"rel"`
			} `json:"source"`
		} `json:"via"`
		CreatedAt          time.Time     `json:"created_at"`
		UpdatedAt          time.Time     `json:"updated_at"`
		GeneratedTimestamp int           `json:"generated_timestamp"`
		Type               string        `json:"type"`
		Subject            string        `json:"subject"`
		RawSubject         string        `json:"raw_subject"`
		Description        string        `json:"description"`
		Priority           string        `json:"priority"`
		Status             string        `json:"status"`
		Recipient          interface{}   `json:"recipient"`
		RequesterId        int64         `json:"requester_id"`
		SubmitterId        int64         `json:"submitter_id"`
		AssigneeId         int64         `json:"assignee_id"`
		OrganizationId     int64         `json:"organization_id"`
		GroupId            int64         `json:"group_id"`
		CollaboratorIds    []int64       `json:"collaborator_ids"`
		FollowerIds        []interface{} `json:"follower_ids"`
		EmailCcIds         []int64       `json:"email_cc_ids"`
		ForumTopicId       interface{}   `json:"forum_topic_id"`
		ProblemId          interface{}   `json:"problem_id"`
		HasIncidents       bool          `json:"has_incidents"`
		IsPublic           bool          `json:"is_public"`
		DueAt              interface{}   `json:"due_at"`
		Tags               []string      `json:"tags"`
		CustomFields       []struct {
			Id    int64   `json:"id"`
			Value *string `json:"value"`
		} `json:"custom_fields"`
		SatisfactionRating struct {
			Score string `json:"score"`
		} `json:"satisfaction_rating"`
		SharingAgreementIds []interface{} `json:"sharing_agreement_ids"`
		CustomStatusId      int           `json:"custom_status_id"`
		EncodedId           string        `json:"encoded_id"`
		Fields              []struct {
			Id    int64   `json:"id"`
			Value *string `json:"value"`
		} `json:"fields"`
		FollowupIds          []interface{} `json:"followup_ids"`
		BrandId              int           `json:"brand_id"`
		AllowChannelback     bool          `json:"allow_channelback"`
		AllowAttachments     bool          `json:"allow_attachments"`
		FromMessagingChannel bool          `json:"from_messaging_channel"`
	} `json:"ticket"`
}

type TicketComments struct {
	Comments []struct {
		Id          int64         `json:"id"`
		Type        string        `json:"type"`
		AuthorId    int64         `json:"author_id"`
		Body        string        `json:"body"`
		HtmlBody    string        `json:"html_body"`
		PlainBody   string        `json:"plain_body"`
		Public      bool          `json:"public"`
		Attachments []interface{} `json:"attachments"`
		AuditId     int64         `json:"audit_id"`
		Via         struct {
			Channel string `json:"channel"`
			Source  struct {
				From struct {
					Address            string   `json:"address,omitempty"`
					Name               string   `json:"name,omitempty"`
					OriginalRecipients []string `json:"original_recipients,omitempty"`
				} `json:"from"`
				To struct {
					Name     string  `json:"name"`
					Address  string  `json:"address"`
					EmailCcs []int64 `json:"email_ccs,omitempty"`
				} `json:"to"`
				Rel interface{} `json:"rel"`
			} `json:"source"`
		} `json:"via"`
		CreatedAt time.Time `json:"created_at"`
		Metadata  struct {
			System struct {
				MessageId           string  `json:"message_id,omitempty"`
				EmailId             string  `json:"email_id,omitempty"`
				RawEmailIdentifier  string  `json:"raw_email_identifier,omitempty"`
				JsonEmailIdentifier string  `json:"json_email_identifier,omitempty"`
				EmlRedacted         bool    `json:"eml_redacted,omitempty"`
				Location            string  `json:"location"`
				Latitude            float64 `json:"latitude"`
				Longitude           float64 `json:"longitude"`
				Client              string  `json:"client,omitempty"`
				IpAddress           string  `json:"ip_address,omitempty"`
			} `json:"system"`
			Custom struct {
			} `json:"custom"`
			SuspensionTypeId interface{} `json:"suspension_type_id"`
		} `json:"metadata"`
	} `json:"comments"`
	Meta struct {
		HasMore      bool   `json:"has_more"`
		AfterCursor  string `json:"after_cursor"`
		BeforeCursor string `json:"before_cursor"`
	} `json:"meta"`
	Links struct {
		Prev string `json:"prev"`
		Next string `json:"next"`
	} `json:"links"`
}

func (c *Client) GetTicket(ctx context.Context, ticketId int64) (*Ticket, error) {
	url := fmt.Sprintf("%s/tickets/%d", c.baseUrl, ticketId)
	t := &Ticket{}

	if err := c.apiRequest(ctx, "GET", url, nil, &t); err != nil {
		return nil, fmt.Errorf("an error occured getting the ticket: %w", err)
	}

	return t, nil
}

func (c *Client) GetAllTicketComments(ctx context.Context, ticketId int64) (*TicketComments, error) {
	initialUrl := fmt.Sprintf("%s/tickets/%d/comments.json?page[size]=100", c.baseUrl, ticketId)
	allComments := &TicketComments{}
	currentPage := &TicketComments{}

	if err := c.apiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting initial ticket comments: %w", err)
	}

	// Append the first page of comments to the allComments slice
	allComments.Comments = append(allComments.Comments, currentPage.Comments...)

	for currentPage.Meta.HasMore {
		nextPage := &TicketComments{}
		log.Printf("Next page: %s", currentPage.Links.Next)
		if err := c.apiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting next page of ticket comments: %w", err)
		}

		allComments.Comments = append(allComments.Comments, nextPage.Comments...)
		currentPage = nextPage

	}

	return allComments, nil
}
