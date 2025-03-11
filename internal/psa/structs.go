package psa

import "time"

type Company struct {
	Id                    int            `json:"id"`
	Identifier            string         `json:"identifier"`
	Name                  string         `json:"name"`
	Status                Status         `json:"status"`
	AddressLine1          string         `json:"addressLine1"`
	AddressLine2          string         `json:"addressLine2"`
	City                  string         `json:"city"`
	Zip                   string         `json:"zip"`
	Country               Country        `json:"country"`
	PhoneNumber           string         `json:"phoneNumber"`
	FaxNumber             string         `json:"faxNumber"`
	Website               string         `json:"website"`
	Territory             Territory      `json:"territory"`
	DefaultContact        DefaultContact `json:"defaultContact"`
	DateAcquired          time.Time      `json:"dateAcquired"`
	AnnualRevenue         float64        `json:"annualRevenue"`
	LeadFlag              bool           `json:"leadFlag"`
	UnsubscribeFlag       bool           `json:"unsubscribeFlag"`
	InvoiceToEmailAddress string         `json:"invoiceToEmailAddress"`
	DeletedFlag           bool           `json:"deletedFlag"`
	MobileGuid            string         `json:"mobileGuid"`
	IsVendorFlag          bool           `json:"isVendorFlag"`
	Types                 []Type         `json:"types"`
	Site                  Site           `json:"site"`
}

type Contact struct {
	Id                 int                 `json:"id"`
	FirstName          string              `json:"firstName"`
	LastName           string              `json:"lastName"`
	Company            Company             `json:"company"`
	Site               Site                `json:"site"`
	InactiveFlag       bool                `json:"inactiveFlag"`
	Title              string              `json:"title,omitempty"`
	MarriedFlag        bool                `json:"marriedFlag"`
	ChildrenFlag       bool                `json:"childrenFlag"`
	UnsubscribeFlag    bool                `json:"unsubscribeFlag"`
	MobileGuid         string              `json:"mobileGuid"`
	DefaultPhoneType   string              `json:"defaultPhoneType,omitempty"`
	DefaultPhoneNbr    string              `json:"defaultPhoneNbr,omitempty"`
	DefaultBillingFlag bool                `json:"defaultBillingFlag"`
	DefaultFlag        bool                `json:"defaultFlag"`
	CompanyLocation    CompanyLocation     `json:"companyLocation"`
	CommunicationItems []CommunicationItem `json:"communicationItems"`
	Types              []interface{}       `json:"types"`
}

type Ticket struct {
	Id                         int             `json:"id"`
	Summary                    string          `json:"summary"`
	RecordType                 string          `json:"recordType"`
	Board                      Board           `json:"board"`
	Status                     Status          `json:"status"`
	Company                    Company         `json:"company"`
	Site                       Site            `json:"site"`
	SiteName                   string          `json:"siteName"`
	AddressLine1               string          `json:"addressLine1"`
	AddressLine2               string          `json:"addressLine2"`
	City                       string          `json:"city"`
	Contact                    Contact         `json:"contact"`
	ContactName                string          `json:"contactName"`
	ContactPhoneNumber         string          `json:"contactPhoneNumber"`
	ContactEmailAddress        string          `json:"contactEmailAddress"`
	Type                       Type            `json:"type"`
	Team                       Team            `json:"team"`
	Owner                      Owner           `json:"owner"`
	Priority                   Priority        `json:"priority"`
	ServiceLocation            ServiceLocation `json:"serviceLocation"`
	Source                     Source          `json:"source"`
	RequiredDate               time.Time       `json:"requiredDate"`
	AgreementType              string          `json:"agreementType"`
	Severity                   string          `json:"severity"`
	Impact                     string          `json:"impact"`
	AllowAllClientsPortalView  bool            `json:"allowAllClientsPortalView"`
	CustomerUpdatedFlag        bool            `json:"customerUpdatedFlag"`
	AutomaticEmailContactFlag  bool            `json:"automaticEmailContactFlag"`
	AutomaticEmailResourceFlag bool            `json:"automaticEmailResourceFlag"`
	AutomaticEmailCcFlag       bool            `json:"automaticEmailCcFlag"`
	ClosedFlag                 bool            `json:"closedFlag"`
	ActualHours                float64         `json:"actualHours"`
	Approved                   bool            `json:"approved"`
	EstimatedExpenseCost       float64         `json:"estimatedExpenseCost"`
	EstimatedExpenseRevenue    float64         `json:"estimatedExpenseRevenue"`
	EstimatedProductCost       float64         `json:"estimatedProductCost"`
	EstimatedProductRevenue    float64         `json:"estimatedProductRevenue"`
	EstimatedTimeCost          float64         `json:"estimatedTimeCost"`
	EstimatedTimeRevenue       float64         `json:"estimatedTimeRevenue"`
	BillingMethod              string          `json:"billingMethod"`
	SubBillingMethod           string          `json:"subBillingMethod"`
	ResolveMinutes             int             `json:"resolveMinutes"`
	ResPlanMinutes             int             `json:"resPlanMinutes"`
	RespondMinutes             int             `json:"respondMinutes"`
	IsInSla                    bool            `json:"isInSla"`
	HasChildTicket             bool            `json:"hasChildTicket"`
	HasMergedChildTicketFlag   bool            `json:"hasMergedChildTicketFlag"`
	BillTime                   string          `json:"billTime"`
	BillExpenses               string          `json:"billExpenses"`
	BillProducts               string          `json:"billProducts"`
	Location                   Location        `json:"location"`
	Department                 Department      `json:"department"`
	EscalationLevel            int             `json:"escalationLevel"`
	CustomFields               []CustomField   `json:"customFields"`
}

type Board struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type CommunicationItem struct {
	Id                int                   `json:"id"`
	Type              CommunicationItemType `json:"type"`
	Value             string                `json:"value"`
	DefaultFlag       bool                  `json:"defaultFlag"`
	Domain            string                `json:"domain,omitempty"`
	CommunicationType string                `json:"communicationType"`
}

type CommunicationItemType struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type CompanyLocation struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Country struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type CustomField struct {
	Id               int    `json:"id"`
	Caption          string `json:"caption"`
	Type             string `json:"type"`
	EntryMethod      string `json:"entryMethod"`
	NumberOfDecimals int    `json:"numberOfDecimals"`
	ConnectWiseId    string `json:"connectWiseId"`
}

type DefaultContact struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Department struct {
	Id         int    `json:"id"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
}

type Location struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Owner struct {
	Id         int    `json:"id"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
}

type Priority struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Sort  int    `json:"sort"`
	Level string `json:"level"`
}

type ServiceLocation struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Site struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Source struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Status struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Sort int    `json:"Sort"`
}

type Team struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Territory struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Type struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
