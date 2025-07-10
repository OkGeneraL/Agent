# 🚀 SuperAgent PaaS Admin Panel

Enterprise-grade admin panel for managing the SuperAgent Platform-as-a-Service (PaaS) business. Built with Next.js 15, TypeScript, Tailwind CSS, and shadcn/ui.

## ✨ Features

### 🏠 Dashboard Overview
- **Real-time Metrics**: Live statistics for customers, deployments, servers, and revenue
- **Revenue Analytics**: Monthly revenue tracking with year-over-year comparisons
- **Activity Feed**: Recent platform events and system notifications
- **System Health**: Server cluster monitoring and status indicators
- **Quick Actions**: Fast access to common admin tasks

### 👥 Customer Management
- **Customer Directory**: Comprehensive list with search and filtering
- **Subscription Management**: Plan upgrades, downgrades, and billing
- **Usage Analytics**: Per-customer resource utilization tracking
- **Support Integration**: Direct access to customer support tools

### 📦 Application Management
- **App Catalog**: Manage marketplace applications and publishers
- **Approval Workflow**: Review and approve new applications
- **Version Control**: Track application versions and deployments
- **Analytics**: Application usage and popularity metrics

### 🚀 Deployment Management
- **Live Monitoring**: Real-time deployment status and health checks
- **Log Streaming**: Live application and build logs
- **Environment Management**: Production, staging, and development environments
- **Automated Scaling**: Auto-scaling configuration and monitoring

### 🖥️ Server Management
- **Cluster Overview**: Visual server topology and health monitoring
- **Resource Allocation**: CPU, memory, and storage management
- **Load Balancing**: Distribution and failover configuration
- **Maintenance Mode**: Scheduled maintenance and updates

### 🌐 Domain & SSL Management
- **Domain Configuration**: Custom domain setup and verification
- **SSL Certificates**: Automatic certificate provisioning and renewal
- **DNS Management**: DNS record configuration and validation
- **Subdomain Management**: Automated subdomain provisioning

### 💳 Billing & Finance
- **Revenue Dashboard**: Financial metrics and trending
- **Invoice Management**: Automated billing and payment processing
- **Plan Management**: Subscription tiers and pricing configuration
- **Financial Reports**: Detailed revenue and usage reports

### 📊 Analytics & Reporting
- **Business Intelligence**: Customer growth and churn analysis
- **Usage Analytics**: Platform utilization and performance metrics
- **Custom Reports**: Flexible reporting with export capabilities
- **Revenue Analytics**: Financial performance tracking

### 🔒 Security & Compliance
- **Admin User Management**: Role-based access control
- **Audit Logging**: Complete activity tracking and compliance
- **Security Monitoring**: Threat detection and incident response
- **Compliance Tools**: GDPR, SOC2, and regulatory compliance

### ⚙️ Settings & Configuration
- **Platform Settings**: Global configuration management
- **Integration Management**: Third-party service integrations
- **Notification Settings**: Alert and notification configuration
- **API Management**: API endpoint and webhook management

## 🛠️ Tech Stack

- **Framework**: Next.js 15 with App Router
- **Language**: TypeScript
- **Styling**: Tailwind CSS 4
- **UI Components**: shadcn/ui
- **Charts**: Recharts
- **Icons**: Lucide React
- **Authentication**: Supabase Auth
- **Database**: Supabase (PostgreSQL)
- **State Management**: Zustand
- **Form Handling**: React Hook Form + Zod
- **Theme**: Dark/Light mode support

## 🚀 Quick Start

### Prerequisites
- Node.js 18+ 
- npm or yarn
- Supabase account (for authentication and database)

### Installation

1. **Clone and navigate to the project**:
   ```bash
   cd adminpanel/admin
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Environment Setup**:
   ```bash
   cp .env.example .env.local
   ```

4. **Configure environment variables** in `.env.local`:
   ```env
   # Supabase Configuration
   NEXT_PUBLIC_SUPABASE_URL=your_supabase_url
   NEXT_PUBLIC_SUPABASE_ANON_KEY=your_supabase_anon_key
   SUPABASE_SERVICE_ROLE_KEY=your_service_role_key
   
   # SuperAgent API Configuration
   NEXT_PUBLIC_SUPERAGENT_API_URL=http://localhost:8080/api/v1
   SUPERAGENT_API_TOKEN=your_admin_api_token
   
   # Security
   NEXTAUTH_SECRET=your_nextauth_secret
   ENCRYPTION_KEY=your_encryption_key
   
   # Features
   NEXT_PUBLIC_ENABLE_ANALYTICS=true
   NEXT_PUBLIC_ENABLE_BILLING=true
   NEXT_PUBLIC_ENABLE_NOTIFICATIONS=true
   ```

5. **Run the development server**:
   ```bash
   npm run dev
   ```

6. **Open your browser**:
   Navigate to [http://localhost:3000](http://localhost:3000)

## 📁 Project Structure

```
adminpanel/admin/
├── app/                          # Next.js 15 App Router
│   ├── (auth)/                   # Authentication routes
│   ├── (dashboard)/              # Protected dashboard routes
│   │   ├── dashboard/            # Main dashboard
│   │   │   ├── customers/        # Customer management
│   │   │   ├── applications/     # App management
│   │   │   ├── deployments/      # Deployment management
│   │   │   ├── servers/          # Server management
│   │   │   ├── domains/          # Domain management
│   │   │   ├── billing/          # Billing management
│   │   │   ├── analytics/        # Analytics dashboard
│   │   │   ├── security/         # Security management
│   │   │   ├── settings/         # Configuration
│   │   │   └── support/          # Support tools
│   │   └── layout.tsx            # Dashboard layout
│   ├── api/                      # API routes
│   ├── globals.css               # Global styles
│   └── layout.tsx                # Root layout
├── components/                   # React components
│   ├── ui/                       # shadcn/ui components
│   ├── layout/                   # Layout components
│   ├── dashboard/                # Dashboard components
│   ├── customers/                # Customer components
│   ├── charts/                   # Chart components
│   └── providers/                # Context providers
├── lib/                          # Utility functions
├── hooks/                        # Custom React hooks
├── types/                        # TypeScript definitions
├── server/                       # Server actions
├── config/                       # Configuration files
└── utils/                        # Helper utilities
```

## 🎨 UI Components

The admin panel uses shadcn/ui components for a consistent and accessible design:

- **Data Display**: Tables, cards, badges, charts
- **Navigation**: Sidebar, breadcrumbs, pagination
- **Forms**: Inputs, selects, checkboxes, radio groups
- **Feedback**: Alerts, toasts, loading states
- **Overlays**: Modals, dropdowns, tooltips
- **Layout**: Grid system, spacing utilities

## 🔐 Authentication & Security

- **Supabase Auth**: Secure authentication with row-level security
- **Role-based Access**: Admin, super admin, and support roles
- **Session Management**: Automatic token refresh and logout
- **Audit Logging**: Complete activity tracking
- **API Security**: Rate limiting and request validation

## 📊 Data Management

- **Real-time Updates**: Live data synchronization
- **Caching Strategy**: Intelligent data caching
- **Error Handling**: Comprehensive error boundaries
- **Loading States**: Skeleton loaders and spinners
- **Optimistic Updates**: Immediate UI feedback

## 🌙 Theme Support

- **Dark/Light Mode**: System preference detection
- **Custom Themes**: Configurable color schemes
- **Accessibility**: WCAG 2.1 compliance
- **Responsive Design**: Mobile-first approach

## 🚀 Deployment

### Production Build
```bash
npm run build
npm start
```

### Docker Deployment
```bash
docker build -t superagent-admin .
docker run -p 3000:3000 superagent-admin
```

### Vercel Deployment
```bash
vercel deploy
```

## 🤝 API Integration

The admin panel integrates with the SuperAgent API for:

- **Customer Management**: CRUD operations
- **Application Lifecycle**: Deployment and monitoring
- **Server Management**: Cluster orchestration
- **Billing Integration**: Payment processing
- **Analytics Data**: Metrics collection

## 📈 Performance

- **Code Splitting**: Automatic route-based splitting
- **Image Optimization**: Next.js image optimization
- **Bundle Analysis**: Build size monitoring
- **Performance Monitoring**: Core Web Vitals tracking

## 🧪 Testing

```bash
# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Run E2E tests
npm run test:e2e
```

## 🔧 Development

### Adding New Pages
1. Create route in `app/(dashboard)/dashboard/`
2. Add navigation item to `components/layout/sidebar.tsx`
3. Implement page component with proper TypeScript types

### Adding New Components
1. Create component in appropriate folder
2. Export from index file
3. Add to Storybook (if applicable)

### Database Schema
The admin panel expects the following Supabase tables:
- `customers` - Customer information
- `applications` - Application catalog
- `deployments` - Deployment records
- `servers` - Server cluster data
- `domains` - Domain configurations
- `invoices` - Billing information

## 📄 License

This project is licensed under the MIT License.

## 🆘 Support

For support and questions:
- 📧 Email: admin-support@superagent.dev
- 📖 Documentation: [Admin Panel Docs](https://docs.superagent.dev/admin)
- 🐛 Issues: [GitHub Issues](https://github.com/superagent/admin-panel/issues)

---

**Built with ❤️ for the SuperAgent PaaS Platform**
