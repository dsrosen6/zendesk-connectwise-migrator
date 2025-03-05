package cw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Ticket struct {
	Id         int    `json:"id"`
	Summary    string `json:"summary"`
	RecordType string `json:"recordType"`
	Board      struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			BoardHref string `json:"board_href"`
		} `json:"_info"`
	} `json:"board"`
	Status struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Sort int    `json:"Sort"`
		Info struct {
			StatusHref string `json:"status_href"`
		} `json:"_info"`
	} `json:"status"`
	WorkRole struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			WorkRoleHref string `json:"workRole_href"`
		} `json:"_info"`
	} `json:"workRole"`
	WorkType struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			WorkTypeHref string `json:"workType_href"`
		} `json:"_info"`
	} `json:"workType"`
	Company struct {
		Id         int    `json:"id"`
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
		Info       struct {
			CompanyHref string `json:"company_href"`
			MobileGuid  string `json:"mobileGuid"`
		} `json:"_info"`
	} `json:"company"`
	Site struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			SiteHref   string `json:"site_href"`
			MobileGuid string `json:"mobileGuid"`
		} `json:"_info"`
	} `json:"site"`
	SiteName     string `json:"siteName"`
	AddressLine1 string `json:"addressLine1"`
	AddressLine2 string `json:"addressLine2"`
	City         string `json:"city"`
	Country      struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			CountryHref string `json:"country_href"`
		} `json:"_info"`
	} `json:"country"`
	Contact struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			MobileGuid  string `json:"mobileGuid"`
			ContactHref string `json:"contact_href"`
		} `json:"_info"`
	} `json:"contact"`
	ContactName         string `json:"contactName"`
	ContactPhoneNumber  string `json:"contactPhoneNumber"`
	ContactEmailAddress string `json:"contactEmailAddress"`
	Type                struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			TypeHref string `json:"type_href"`
		} `json:"_info"`
	} `json:"type"`
	Team struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			TeamHref string `json:"team_href"`
		} `json:"_info"`
	} `json:"team"`
	Owner struct {
		Id         int    `json:"id"`
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
		Info       struct {
			MemberHref string `json:"member_href"`
			ImageHref  string `json:"image_href"`
		} `json:"_info"`
	} `json:"owner"`
	Priority struct {
		Id    int    `json:"id"`
		Name  string `json:"name"`
		Sort  int    `json:"sort"`
		Level string `json:"level"`
		Info  struct {
			PriorityHref string `json:"priority_href"`
			ImageHref    string `json:"image_href"`
		} `json:"_info"`
	} `json:"priority"`
	ServiceLocation struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			LocationHref string `json:"location_href"`
		} `json:"_info"`
	} `json:"serviceLocation"`
	Source struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			SourceHref string `json:"source_href"`
		} `json:"_info"`
	} `json:"source"`
	RequiredDate time.Time `json:"requiredDate"`
	Agreement    struct {
		Id             int    `json:"id"`
		Name           string `json:"name"`
		Type           string `json:"type"`
		ChargeFirmFlag bool   `json:"chargeFirmFlag"`
		Info           struct {
			AgreementHref string `json:"agreement_href"`
			TypeId        string `json:"typeId"`
		} `json:"_info"`
	} `json:"agreement"`
	AgreementType              string  `json:"agreementType"`
	Severity                   string  `json:"severity"`
	Impact                     string  `json:"impact"`
	AllowAllClientsPortalView  bool    `json:"allowAllClientsPortalView"`
	CustomerUpdatedFlag        bool    `json:"customerUpdatedFlag"`
	AutomaticEmailContactFlag  bool    `json:"automaticEmailContactFlag"`
	AutomaticEmailResourceFlag bool    `json:"automaticEmailResourceFlag"`
	AutomaticEmailCcFlag       bool    `json:"automaticEmailCcFlag"`
	ClosedFlag                 bool    `json:"closedFlag"`
	ActualHours                float64 `json:"actualHours"`
	Approved                   bool    `json:"approved"`
	EstimatedExpenseCost       float64 `json:"estimatedExpenseCost"`
	EstimatedExpenseRevenue    float64 `json:"estimatedExpenseRevenue"`
	EstimatedProductCost       float64 `json:"estimatedProductCost"`
	EstimatedProductRevenue    float64 `json:"estimatedProductRevenue"`
	EstimatedTimeCost          float64 `json:"estimatedTimeCost"`
	EstimatedTimeRevenue       float64 `json:"estimatedTimeRevenue"`
	BillingMethod              string  `json:"billingMethod"`
	SubBillingMethod           string  `json:"subBillingMethod"`
	ResolveMinutes             int     `json:"resolveMinutes"`
	ResPlanMinutes             int     `json:"resPlanMinutes"`
	RespondMinutes             int     `json:"respondMinutes"`
	IsInSla                    bool    `json:"isInSla"`
	HasChildTicket             bool    `json:"hasChildTicket"`
	HasMergedChildTicketFlag   bool    `json:"hasMergedChildTicketFlag"`
	BillTime                   string  `json:"billTime"`
	BillExpenses               string  `json:"billExpenses"`
	BillProducts               string  `json:"billProducts"`
	Location                   struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			LocationHref string `json:"location_href"`
		} `json:"_info"`
	} `json:"location"`
	Department struct {
		Id         int    `json:"id"`
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
		Info       struct {
			DepartmentHref string `json:"department_href"`
		} `json:"_info"`
	} `json:"department"`
	MobileGuid string `json:"mobileGuid"`
	Sla        struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			SlaHref string `json:"sla_href"`
		} `json:"_info"`
	} `json:"sla"`
	RequestForChangeFlag bool `json:"requestForChangeFlag"`
	Currency             struct {
		Id                      int    `json:"id"`
		Symbol                  string `json:"symbol"`
		CurrencyCode            string `json:"currencyCode"`
		DecimalSeparator        string `json:"decimalSeparator"`
		NumberOfDecimals        int    `json:"numberOfDecimals"`
		ThousandsSeparator      string `json:"thousandsSeparator"`
		NegativeParenthesesFlag bool   `json:"negativeParenthesesFlag"`
		DisplaySymbolFlag       bool   `json:"displaySymbolFlag"`
		CurrencyIdentifier      string `json:"currencyIdentifier"`
		DisplayIdFlag           bool   `json:"displayIdFlag"`
		RightAlign              bool   `json:"rightAlign"`
		Name                    string `json:"name"`
		Info                    struct {
			CurrencyHref string `json:"currency_href"`
		} `json:"_info"`
	} `json:"currency"`
	Info struct {
		LastUpdated         time.Time `json:"lastUpdated"`
		UpdatedBy           string    `json:"updatedBy"`
		DateEntered         time.Time `json:"dateEntered"`
		EnteredBy           string    `json:"enteredBy"`
		ActivitiesHref      string    `json:"activities_href"`
		ScheduleentriesHref string    `json:"scheduleentries_href"`
		DocumentsHref       string    `json:"documents_href"`
		ConfigurationsHref  string    `json:"configurations_href"`
		TasksHref           string    `json:"tasks_href"`
		NotesHref           string    `json:"notes_href"`
		ProductsHref        string    `json:"products_href"`
		TimeentriesHref     string    `json:"timeentries_href"`
		ExpenseEntriesHref  string    `json:"expenseEntries_href"`
	} `json:"_info"`
	EscalationStartDateUTC  time.Time `json:"escalationStartDateUTC"`
	EscalationLevel         int       `json:"escalationLevel"`
	MinutesBeforeWaiting    int       `json:"minutesBeforeWaiting"`
	RespondedSkippedMinutes int       `json:"respondedSkippedMinutes"`
	ResplanSkippedMinutes   int       `json:"resplanSkippedMinutes"`
	RespondedHours          float64   `json:"respondedHours"`
	ResplanHours            float64   `json:"resplanHours"`
	ResolutionHours         float64   `json:"resolutionHours"`
	MinutesWaiting          int       `json:"minutesWaiting"`
	CustomFields            []struct {
		Id               int    `json:"id"`
		Caption          string `json:"caption"`
		Type             string `json:"type"`
		EntryMethod      string `json:"entryMethod"`
		NumberOfDecimals int    `json:"numberOfDecimals"`
		ConnectWiseId    string `json:"connectWiseId"`
	} `json:"customFields"`
}

func (c *Client) GetTicket(ctx context.Context, ticketId int) (Ticket, error) {
	url := fmt.Sprintf("%s/service/tickets/%d", baseUrl, ticketId)
	t := &Ticket{}

	if err := c.apiRequest(ctx, "GET", url, nil, &t); err != nil {
		return Ticket{}, fmt.Errorf("an error occured getting the ticket: %w", err)
	}

	return *t, nil
}

func (c *Client) PostTicket(ctx context.Context, ticket Ticket) (Ticket, error) {
	url := fmt.Sprintf("%s/service/tickets", baseUrl)

	ticketBytes, err := json.Marshal(ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("an error occured marshaling the ticket to json: %w", err)
	}

	body := bytes.NewReader(ticketBytes)
	t := &Ticket{}

	if err := c.apiRequest(ctx, "POST", url, body, t); err != nil {
		return Ticket{}, fmt.Errorf("an error occured posting the ticket: %w", err)
	}

	return *t, nil
}
