# Zendesk to ConnectWise Migration Utility
This utility is designed to migrate users and tickets from Zendesk to ConnectWise PSA (formerly Manage). It is run within the terminal and is customizable depending on what you need in your migration. Primarily, ticket sourcing is Zendesk tag-based, and from there you can set date ranges per tag for what tickets to migrate.

The terminal interface is a bit rough around the edges and is definitely more function over form! All visual elements are powered by Charm - mostly, their [Bubble Tea framework.](https://github.com/charmbracelet/bubbletea)

## Getting Started
This guide assumes you have a baseline understanding of the Zendesk and ConnectWise PSA APIs. You will need to set up an API key for Zendesk, and create an API member/Client ID for ConnectWise. 
1. Install the binary and run it on your terminal.