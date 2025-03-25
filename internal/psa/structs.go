package psa

type Company struct {
	Id           int        `json:"id,omitempty"`
	Identifier   string     `json:"identifier,omitempty"`
	Name         string     `json:"name,omitempty"`
	AddressLine1 string     `json:"addressLine1,omitempty"`
	AddressLine2 string     `json:"addressLine2,omitempty"`
	City         string     `json:"city,omitempty"`
	Zip          string     `json:"zip,omitempty"`
	Country      *Country   `json:"country,omitempty"`
	Territory    *Territory `json:"territory,omitempty"`
	Site         *Site      `json:"site,omitempty"`
}

type ContactPostBody struct {
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Company   struct {
		Id int `json:"id,omitempty"`
	} `json:"company"`
	CommunicationItems []CommunicationItem `json:"communicationItemsm,omitempty"`
}

type Contact struct {
	Id                 int                 `json:"id,omitempty"`
	FirstName          string              `json:"firstName,omitempty"`
	LastName           string              `json:"lastName,omitempty"`
	CommunicationItems []CommunicationItem `json:"communicationItems,omitempty"`
}

type Ticket struct {
	Id                      int           `json:"id,omitempty"`
	Summary                 string        `json:"summary,omitempty"`
	InitialDescription      string        `json:"initialDescription,omitempty"`
	InitialInternalAnalysis string        `json:"InitialInternalAnalysis,omitempty"`
	Board                   *Board        `json:"board,omitempty"`
	Status                  *Status       `json:"status,omitempty"`
	Company                 *Company      `json:"company,omitempty"`
	Contact                 *Contact      `json:"contact,omitempty"`
	Owner                   *Owner        `json:"owner,omitempty"`
	CustomFields            []CustomField `json:"customFields,omitempty"`
}

type Board struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type CommunicationItem struct {
	Id                int                   `json:"id,omitempty"`
	Type              CommunicationItemType `json:"type,omitempty"`
	Value             string                `json:"value,omitempty"`
	DefaultFlag       bool                  `json:"defaultFlag,omitempty"`
	Domain            string                `json:"domain,omitempty"`
	CommunicationType string                `json:"communicationType,omitempty"`
}

type CommunicationItemType struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type CompanyLocation struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Country struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type CustomField struct {
	Id               int    `json:"id,omitempty"`
	Caption          string `json:"caption,omitempty"`
	Type             string `json:"type,omitempty"`
	EntryMethod      string `json:"entryMethod,omitempty"`
	NumberOfDecimals int    `json:"numberOfDecimals,omitempty"`
	ConnectWiseId    string `json:"connectWiseId,omitempty"`
}

type Owner struct {
	Id         int    `json:"id,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	Name       string `json:"name,omitempty"`
}

type Site struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Status struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Territory struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type BoardType struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Member struct {
	Id           int    `json:"id,omitempty"`
	Identifier   string `json:"identifier,omitempty"`
	ClientId     string `json:"clientId,omitempty"`
	FirstName    string `json:"firstName,omitempty"`
	OfficeEmail  string `json:"officeEmail,omitempty"`
	DefaultEmail string `json:"defaultEmail,omitempty"`
	PrimaryEmail string `json:"primaryEmail,omitempty"`
}

type TicketNote struct {
	Id                    int      `json:"id,omitempty"`
	TicketId              int      `json:"ticketId,omitempty"`
	Text                  string   `json:"text,omitempty"`
	DetailDescriptionFlag bool     `json:"detailDescriptionFlag,omitempty"`
	InternalAnalysisFlag  bool     `json:"internalAnalysisFlag,omitempty"`
	ResolutionFlag        bool     `json:"resolutionFlag,omitempty"`
	IssueFlag             bool     `json:"issueFlag,omitempty"`
	Contact               *Contact `json:"contact,omitempty"`
	CreatedBy             string   `json:"createdBy,omitempty"`
	InternalFlag          bool     `json:"internalFlag,omitempty"`
	ExternalFlag          bool     `json:"externalFlag,omitempty"`
	Member                *Member  `json:"member,omitempty"`
}
