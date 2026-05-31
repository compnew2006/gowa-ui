# Kingmaster — Minimum Viable Product Requirements

**Version:** 3.0 — Feature Inventory (Minimum Features Only)
**Date:** 2026-05-29

---

## 1. User Authentication & Access Control

### 1.1 Registration & Login
- Email-based registration with first name, last name, phone, password
- Login with email + password
- Password hashing with secure algorithms
- Session management with secure cookies
- "Remember me" capability
- Logout with session destruction

### 1.2 OTP Verification
- One-time password sent to user email after registration
- OTP expiry timer and regeneration
- Account activation only after OTP verification
- Resend OTP functionality

### 1.3 Password Recovery
- "Forgot password" flow via email
- Reset password with time-limited token
- Change password for authenticated users (requires current password)

### 1.4 Role-Based Access
- Two roles: **Admin** and **User**
- Admin can access all management screens, user lists, and platform-wide analytics
- Users can only access their own data, campaigns, and referrals
- Admin can switch into any user account and return to admin

### 1.5 Referral-Based Registration
- Users can register using a referral link/code from another user
- Referral code is tracked and stored in the MLM tree on registration
- Referral code verification before registration completes

---

## 2. User Dashboard

### 2.1 Home Dashboard
- Welcome section with user name and profile avatar
- Key stats at a glance: total campaigns sent, total messages delivered, current points balance, package name and expiry date
- Weekly campaign activity chart (messages sent per day)
- Platform breakdown chart (WhatsApp vs Facebook vs Instagram usage)
- Recent activity log (last N actions by the user)
- Quick-access action cards (create campaign, manage contacts, etc.)

### 2.2 Profile Management
- View and edit first name, last name, email, phone
- Upload and change profile avatar image
- Set personal timezone
- View job title and birth date
- Membership expiry date display

---

## 3. Multi-Platform Account Management

### 3.1 Account Connection (WhatsApp)
- Add WhatsApp accounts via QR code scan or session cookies
- Store session data securely per account
- View account connection status (active / inactive / closed)
- Delete or disconnect accounts
- Per-account unique identifier tracking

### 3.2 Account Connection (Facebook)
- Add Facebook accounts via cookie/data authentication
- Manage Facebook page accounts (associate pages to user)
- Store and refresh OAuth tokens for pages
- View page name, page ID, and token status

### 3.3 Account Connection (Instagram)
- Add Instagram business accounts
- Store Instagram session data
- View connection status

### 3.4 Unified Account List
- Single view listing all connected accounts across all platforms
- Filter by platform (WhatsApp, Facebook, Instagram)
- Filter by status (active, inactive, closed)
- Account usage count and limits per package

---

## 4. WhatsApp Campaigns & Messaging

### 4.1 Bulk Message Campaigns
- Create named campaigns with a list of target phone numbers
- Support for text messages, image messages, video messages, PDF documents
- Message content drawn from the content library (see section 8)
- Campaign states: pending → running → paused → stopped → finished
- Real-time counters: total targets, successfully sent, failed
- Pause, resume, and stop running campaigns
- Delete single or all campaigns
- Campaign scheduling with configurable send intervals (delay between messages)
- Speed control (normal, fast, etc.)

### 4.2 WhatsApp Group Campaigns
- Send messages to WhatsApp groups
- Select target groups from a searchable group directory
- Track group participant count for reach estimation

### 4.3 WhatsApp Number Filtering
- Upload a list of phone numbers
- System checks which numbers are registered on WhatsApp
- Returns filtered list with status (valid / invalid)
- Export filtered results

### 4.4 WhatsApp Group Extraction
- Extract members from WhatsApp groups
- Extract group metadata (name, participant count, group ID)
- Save extracted groups to user's contact lists

### 4.5 WhatsApp Contact Extraction
- Extract contacts from WhatsApp chats/conversations
- Extract profile information from chat histories

### 4.6 WhatsApp Message Extraction
- Extract chat messages from WhatsApp conversations
- Save message history for analysis or archival

### 4.7 WhatsApp Chat Interface
- Real-time one-on-one chat view for connected WhatsApp accounts
- Send and receive messages inline
- View conversation history
- Read/unread message indicators

### 4.8 WhatsApp Polls
- Create poll messages with multiple choice options
- Generate poll URLs with configurable API integration
- List, search, and manage saved polls

### 4.9 WhatsApp Proxy Support
- Configure proxy settings for WhatsApp connections
- Store proxy configurations per account or globally

---

## 5. AI Chatbot & Flow Builder

### 5.1 Visual Flow Builder
- Drag-and-drop conversation flow designer
- Create nodes for: welcome message, text response, condition check, menu selection
- Connect nodes visually to define conversation paths
- Save and load flows per WhatsApp account

### 5.2 Chatbot Configuration
- Assign a chatbot flow to a specific WhatsApp account
- Enable/disable chatbot per account
- Test chatbot conversations before going live

### 5.3 Auto-Reply Templates
- Define keyword-triggered auto-replies
- Rich media auto-replies (text, image, document)
- Manage template library (create, edit, delete)

---

## 6. Facebook Tools

### 6.1 Facebook Page Search
- Search for Facebook pages by keyword
- Display page name, followers count, page ID
- Save search results for campaign targeting

### 6.2 Facebook Group Search
- Search for Facebook groups by keyword
- Display group name, member count, group link
- Save groups for later campaign use

### 6.3 Facebook People Search
- Search for people on Facebook by keyword
- Display name, profile ID, and available data
- Save people data to campaign databases

### 6.4 Facebook Post Engagement
- Like posts on behalf of connected accounts
- Extract comments from posts
- Extract likes/reactions from posts
- Create posts on connected Facebook pages

### 6.5 Facebook Data Extraction
- Extract user data from Facebook search results
- Store phone numbers, emails, names, locations from public profiles
- Build contact databases from extracted data

### 6.6 Facebook Comment Campaigns
- Create campaigns that post comments on target posts
- Configure comment text, target post URLs, and scheduling
- Track comment campaign success/failure counts

### 6.7 Facebook Page Messaging
- Send messages via Facebook pages
- Create page-based messaging campaigns

### 6.8 Facebook Analytics
- Page-level analytics (followers, engagement metrics)
- Campaign-level performance for Facebook tools

---

## 7. Instagram Tools

### 7.1 Instagram Profile Search
- Search for Instagram users by username or keyword
- Display profile info: username, full name, bio, follower count, verification status
- Search by hashtag
- Search by location

### 7.2 Instagram Data Extraction
- **Extract followers** from target accounts
- **Extract following** lists from target accounts
- **Extract post likers** from specific posts
- **Extract commenters** from specific posts
- **Extract post data** (content, likes, comments, dates)
- **Extract direct messages** from inbox
- **Extract story viewers** from active stories

### 7.3 Instagram Auto-Post
- Schedule and publish posts to Instagram
- Schedule and publish stories to Instagram
- Support image and video uploads

### 7.4 Instagram Follow/Unfollow Tool
- Mass follow users from extracted lists
- Mass unfollow users
- Track follow/unfollow campaign progress

### 7.5 Instagram Mention Tool
- Tag/mention users in posts or comments
- Bulk mention campaigns from extracted user lists

### 7.6 Instagram Direct Messaging
- Send direct messages to Instagram users
- Create DM campaigns with message content and target lists
- Track DM delivery status

### 7.7 Instagram Retargeting
- Re-engage previously extracted users
- Track retargeting campaign status per user
- Build retargeting lists from past campaign data

---

## 8. Content Management System

### 8.1 Content Library
- Create text content items with a name and body
- Character count and word count auto-calculation
- Edit and delete content items
- Bulk delete content
- Search content by name or text

### 8.2 Spintax Support
- Content supports Spintax syntax: `{option1|option2|option3}`
- Each campaign send randomly selects one option per Spintax block
- Creates message variation automatically without manual duplication

### 8.3 Emoji Picker
- Integrated emoji insertion when composing content
- Quick-access emoji panel for common emojis

### 8.4 Content Tags & Categories
- Organize content with tags
- Filter content by tag
- Categorize content for different campaign types

### 8.5 Media Attachment
- Upload images, videos, and PDF files
- Attach media to content items or directly to campaigns
- File management: list, download, delete user files
- File size tracking and storage management

### 8.6 Content Admin Panel
- Admin-managed content messages library
- System-wide content templates available to all users
- Enable/disable content items
- Title, category, and tags for admin content

---

## 9. Contact Management

### 9.1 Contact Lists
- Create named contact lists per platform (WhatsApp, Facebook, Instagram)
- Add contacts manually or via CSV upload
- Store contacts as structured data (name, phone, platform)
- Edit and delete contact lists
- View contact count per list

### 9.2 Contact Import
- Import from CSV files
- Import from text (paste numbers)
- Bulk contact addition with validation

### 9.3 Contact Filtering
- Search contacts by name or number
- Filter by platform
- Filter by list membership
- Count total contacts across all lists

### 9.4 Sending Settings per Contact
- Track which contacts have been sent to (avoid duplicates)
- Reset sending progress for re-campaigns
- Progress indicators per contact list

---

## 10. MLM Referral & Commission System

### 10.1 Referral Tree (Genealogy)
- Each user has a unique referral code
- New users register under a referrer
- System tracks up to 4 levels of referral depth
- Visual MLM tree view showing downline structure
- Expand/collapse tree nodes
- Display referral count per user per level

### 10.2 Commission Calculation
- Automated commission calculation on every package purchase
- Commission rates per level:
  - Level 1 (direct referrer): configurable percentage
  - Level 2: configurable percentage
  - Level 3: configurable percentage
  - Level 4: configurable percentage
- Commission is calculated on the final amount after discounts
- Commission is credited to each upline user's commission wallet

### 10.3 Commission Wallet
- Each user has a commission wallet showing accumulated earnings
- View commission history with timestamps
- Commission balance displayed on dashboard

### 10.4 MLM Settings (Admin)
- Configure commission percentages per level
- Configure MLM-specific package settings (which packages generate commissions)
- Enable/disable MLM features globally

### 10.5 Referral Statistics
- Total referral count per user
- Downline performance metrics
- MLM earnings summary

---

## 11. Wallet & Financial System

### 11.1 Digital Wallet
- Each user has a wallet with balance and points
- View current balance and points total
- Transaction history with date filtering and month filtering
- Paginated transaction list

### 11.2 Money Transfer (Peer-to-Peer)
- Transfer money to another user by email
- Transfer points to another user
- Requires password confirmation for security
- Minimum transfer amount validation
- Transaction record created for both sender and receiver
- CSRF protection on transfers

### 11.3 Wallet OTP Verification
- OTP-based verification for sensitive wallet operations
- OTP sent to registered email
- Time-limited OTP expiry

---

## 12. Points System

### 12.1 Points Balance
- Each user has a points balance
- Points are consumed when sending campaigns (as messaging credits)
- Points displayed on dashboard and profile

### 12.2 Points Packages
- Admin creates points packages with name, price, and point amount
- Users purchase points packages to top up
- Package list with pricing display

### 12.3 Points Purchase
- Purchase flow for points packages
- Payment method selection
- Coupon code application during purchase
- Points credited immediately upon successful purchase

### 12.4 Points Settings (Admin)
- Configure default points allocation per subscription package
- Manage points package catalog (create, edit, delete)
- Set package popularity/featured status

---

## 13. Subscription Packages

### 13.1 Package Catalog
- Multiple subscription tiers (e.g., Basic, Professional, Enterprise)
- Each package defines: name, price, duration in days, features, message limits, account limits
- Package descriptions in multiple languages
- Platform support flags per package (which platforms are included)

### 13.2 Package Purchase
- Users select a package and complete checkout
- Coupon code support at checkout
- Discount application (percentage or fixed amount)
- Package activation on successful payment
- Membership expiry date set based on package duration

### 13.3 Package Management (Admin)
- Create, edit, and delete packages
- Set price, discount, features, and limits
- Mark packages as popular/featured
- Multi-language name and description support
- Package MLM commission settings (which packages generate commissions)

### 13.4 Membership Expiry
- Automatic expiry tracking based on purchase date + package duration
- Users notified of approaching expiry
- Renewal flow to extend membership
- Expired users have restricted access

---

## 14. Coupon / Discount System

### 14.1 Coupon Management (Admin)
- Create coupon codes with: code, type, value, usage limit, expiry date
- Coupon types: discount (percentage), extra time, bonus points, fixed amount off
- Enable/disable coupons
- Track usage count per coupon

### 14.2 Coupon Validation
- Validate coupon code at checkout
- Check expiry date, usage limits, and active status
- Apply discount to final price
- Error messages for invalid/expired/exhausted coupons

### 14.3 Coupon Application
- Apply at package purchase
- Apply at points purchase
- Single coupon per transaction

---

## 15. Product & Order System

### 15.1 Product Catalog
- List products with name, description, price, category
- Support for physical and digital products
- Product images
- Color and size variants
- Stock quantity tracking
- Search and filter products (by category, price range, digital/physical)

### 15.2 Product Management (Admin)
- Create, edit, delete products
- Set price, category, stock, and variants
- Upload product images
- Mark as digital or physical

### 15.3 Order System
- Users place orders for products
- Order status tracking: pending, processing, shipped, delivered, cancelled
- Order details: product, quantity, total, shipping info, phone number
- Order search by ID or phone number
- Date range filtering on orders

### 15.4 Order Management (Admin)
- View all orders across all users
- Update order status
- Process refunds

### 15.5 Order Tracking
- Users can track their order status
- Real-time status updates

---

## 16. Withdrawal System

### 16.1 User Withdrawal Requests
- Users request withdrawal of wallet balance
- Specify withdrawal amount
- Withdrawal status tracking: pending, approved, rejected, processed

### 16.2 Admin Withdrawal Management
- View all withdrawal requests from all users
- Approve or reject withdrawals
- Process withdrawals (mark as completed)
- Add admin notes to withdrawals

### 16.3 System-Wide Financial Records
- Track all financial movements platform-wide
- Filter by date, year, month
- Aggregated financial summaries

---

## 17. Analytics & Statistics

### 17.1 User Analytics
- Total registered users
- Active vs inactive users
- New registrations per time period
- User distribution by country (geographic analytics)
- Birthday alerts for users (today/this month)
- User engagement metrics

### 17.2 Campaign Analytics
- Total campaigns per status
- Message delivery success/failure rates
- Campaign performance over time
- Platform-specific campaign metrics

### 17.3 Revenue Analytics
- Total revenue tracking
- Revenue by package
- Revenue by time period
- Commission payouts summary
- Withdrawal totals

### 17.4 Package Performance Analytics
- Subscription count per package
- Package popularity ranking
- Package renewal rates
- Filtered subscription reports

### 17.5 Data Export
- Export user data to CSV
- Export campaign data to CSV
- Export transaction data
- Export filtered subscription data

### 17.6 Admin Statistics Dashboard
- Platform-wide overview cards (total users, revenue, active campaigns, etc.)
- Geographical distribution chart
- Package distribution chart
- Registration trends chart

---

## 18. Announcement System

### 18.1 Announcements (Admin)
- Create announcements with title and message body
- Enable/disable announcements
- Edit and delete announcements
- Set active/inactive status

### 18.2 Announcements (User)
- View active announcements
- Unread announcement indicator (badge count)
- Mark announcements as read/viewed
- Dismiss announcements

---

## 19. Notification System

### 19.1 In-App Notifications
- Notification bell in top navigation
- Unread notification count badge
- Notification list with timestamps
- Mark individual notifications as read
- Notification types: system alerts, campaign completions, financial events, referral events

### 19.2 Notification Helper
- Helper functions to generate notifications for various events
- Store notifications in database per user
- Retrieve notifications with read/unread filtering

---

## 20. Landing Page

### 20.1 Public Landing Page
- Modern, responsive landing page for non-authenticated visitors
- Hero section with value proposition
- Features section highlighting platform capabilities
- Pricing section showing available packages with real-time pricing
- Discount display (active promotions)
- FAQ section
- Multi-language switching (Arabic / English)
- Registration call-to-action buttons
- "Coming soon" page variant

### 20.2 Landing Page Management
- Dynamic package data pulled from the package catalog
- Discount integration (auto-display active promotions)
- Multiple landing page variants/layouts

---

## 21. Multi-Language (i18n) System

### 21.1 Language Switching
- Toggle between Arabic (RTL) and English (LTR)
- Language preference persisted per user
- Full translation coverage for all UI strings

### 21.2 RTL/LTR Layout
- Automatic layout direction switch based on selected language
- Sidebar position reversal
- Text alignment changes
- Font switching (Cairo for Arabic, Roboto for English)
- Icon and margin adjustments for RTL context

### 21.3 Country Detection
- Auto-detect user country for default language suggestion
- Timezone auto-detection based on country

---

## 22. Sales Target & Customer Tracking

### 22.1 Sales Targets
- Set sales targets (revenue goals)
- Track progress toward targets
- Dashboard visualization of target vs actual

### 22.2 Customer Management
- Add customers with details
- Track customer subscription status
- Renew customer subscriptions
- Delete customer records
- Search and filter customers

---

## 23. Data Extraction Tools

### 23.1 Phone Number Extraction
- Extract phone numbers from Facebook data
- Build targeted contact databases
- Filter extracted data by location, gender, relationship status

### 23.2 Data Search & Filtering
- Search extracted Facebook data by name, phone, email
- Filter by demographic fields (location, gender, work, education)
- Paginated results for large datasets

### 23.3 Retargeting System
- Mark extracted contacts for retargeting campaigns
- Track retargeting status per contact
- Build retargeting lists from previous campaign data

---

## 24. Template System

### 24.1 Message Templates
- Create reusable message templates
- Template types: generic, welcome, follow-up, etc.
- Channel-specific templates (Facebook, WhatsApp, Instagram)
- Template payload stored as structured JSON
- Search and filter templates

### 24.2 Template Management (Admin)
- Admin can create system-wide templates
- Enable/disable templates
- Edit template content and configuration

---

## 25. Sending Settings & Scheduling

### 25.1 Sending Configuration
- Set message sending intervals (delay between messages)
- Configure intervals per platform (WhatsApp, Facebook, Instagram)
- Save, update, and delete sending configurations
- Per-user sending settings

### 25.2 Campaign Scheduling
- Schedule campaigns for future delivery
- Timezone-aware scheduling
- Speed settings per campaign

---

## 26. Security & Privacy

### 26.1 Authentication Security
- Secure password hashing
- CSRF token protection on all state-changing operations
- Rate limiting on sensitive operations (wallet transfers, OTP requests)
- Session-based authentication with secure cookie settings

### 26.2 Data Protection
- SQL injection prevention via prepared statements
- XSS protection via input sanitization
- HTTPS enforcement
- Secure file upload validation (type, size)

### 26.3 Privacy & Terms
- Privacy policy page
- Terms of service page
- Help center / documentation page
- Contact page

### 26.4 Activity Logging
- Log user actions (login, campaign creation, purchases, etc.)
- View activity log per user
- Timestamped log entries

### 26.5 Account Settings
- Security settings page
- Profile editing
- Timezone management
- Account preferences

---

## 27. System Administration

### 27.1 Admin Dashboard
- Platform-wide overview with key metrics
- User management (list, add, edit, delete, activate/deactivate)
- User role management (promote to admin, demote)
- Account switching (admin can log in as any user)
- Return to admin from impersonated session

### 27.2 Campaign Administration
- View all campaigns across all users
- Delete campaigns
- View campaign statistics platform-wide

### 27.3 Content Administration
- Manage system-wide content templates
- Enable/disable content for all users
- Content moderation capabilities

### 27.4 Media Administration
- Upload and manage system media files
- Media library management
- File type categorization

### 27.5 Tools Administration
- Configure platform-wide tool settings
- Manage proxy settings
- Configure platform integrations

### 27.6 Withdrawal Administration
- Process withdrawal requests
- Financial reporting
- System-wide transaction records

### 27.7 Announcements Administration
- Create and manage platform announcements
- Set announcement visibility and scheduling

### 27.8 Analytics Administration
- Full analytics dashboard with all platform metrics
- Export capabilities for all data types
- Filtered views and date-range reporting

---

## 28. File Management

### 28.1 File Upload
- Upload images (campaign media, profile avatars, product images)
- Upload videos
- Upload PDF documents
- File type and size validation
- Original filename preservation

### 28.2 File Library
- List all user files with metadata (name, type, size, upload date)
- Download files
- Delete files
- Search files by name

### 28.3 Media Library (Admin)
- System-wide media management
- Title and metadata for media files
- Categorize by type (image, video, PDF, file)

---

## 29. Proxy Management

### 29.1 Proxy Configuration
- Add proxy settings for platform connections
- Store proxy configurations in JSON format
- Support for HTTP/HTTPS proxies
- Per-campaign or global proxy settings

---

## 30. Search & Discovery (Group Directory)

### 30.1 Group Directory
- Searchable directory of WhatsApp groups
- Group metadata: name, link, country, language, description, category, image
- Full-text search on group names
- Browse by category
- Join group via link

### 30.2 People Search
- Search for people across platforms
- Platform-specific search (Facebook, Instagram)
- Result pagination and filtering
