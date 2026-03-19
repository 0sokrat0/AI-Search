import type { AvatarProps } from '@nuxt/ui'

export type AccountStatus = 'active' | 'authorized' | 'starting' | 'unauthorized' | 'auth_pending' | 'disabled'

export interface TelegramAccount {
  id: string
  phone: string
  session_path: string
  proxy: string
  status: AccountStatus
  qr_url?: string
  waiting_for_password?: boolean
  name: string
  username: string
  avatar_url: string
  created_at: string
  updated_at: string
}

export type UserStatus = 'subscribed' | 'unsubscribed' | 'bounced'
export type SaleStatus = 'paid' | 'failed' | 'refunded'
export type BackendRole = 'super_admin' | 'employee'

export interface BackendError {
  code: string
  message: string
}

export interface BackendResponse<T> {
  success: boolean
  data?: T
  error?: BackendError
}

export interface User {
  id: number
  name: string
  email: string
  avatar?: AvatarProps
  status: UserStatus
  location: string
}

export interface Mail {
  id: number
  signalId: string // реальный UUID из MongoDB (для API вызовов)
  unread?: boolean
  from: User
  telegramUsername?: string
  subject: string
  body: string
  date: string
  leadId?: string | null
  merchantName?: string // имя компании, привязанной к лиду (заполняется локально)
  leadScore?: number | null
  similarityScore?: number | null
  classifiedAsLead?: boolean | null
  semanticDirection?: string | null
  semanticCategory?: 'traders' | 'merchants' | 'ps_offers' | 'noise' | string
  classificationReason?: string | null
  traderScore?: number | null
  merchantScore?: number | null
  processingRequestScore?: number | null
  psOfferScore?: number | null
  noiseScore?: number | null
  primaryLabel?: string | null
  primaryPercent?: number | null
  categoryAssignedAt?: string
  senderTelegramId: number
  isIgnored: boolean
  isTeamMember: boolean
  isSpamSender?: boolean
  isDm: boolean
  otherChatsCount: number
  showMultiAccountBadges?: boolean
  semanticFlags?: string[]
  category?: 'traders' | 'merchants' | 'ps_offers' | 'noise'
  categoryReason?: string
}

export type LeadStatus = 'new' | 'detected' | 'confirmed' | 'controversial' | 'false_positive' | 'contacted' | 'qualified' | 'converted' | 'rejected'
export type LeadPriority = 'low' | 'medium' | 'high' | 'critical'
export type LeadQualificationSource = 'ai_qualified' | 'manual_approved'

export interface Lead {
  id: string
  name: string // senderName
  contact: string // @username or senderID
  avatar?: AvatarProps
  chatTitle: string
  text?: string
  semanticDirection?: string
  semanticCategory?: 'leads' | 'traders' | 'merchants' | 'ps_offers' | 'noise' | string
  merchantId: string
  companyId?: string
  company?: string
  qualificationSource?: LeadQualificationSource | null
  status: LeadStatus
  priority: LeadPriority
  score: number
  geo: string[]
  products: string[]
  userFeedback: boolean | null
  categoryAssignedAt?: string
  signalsCount: number
  lastSeenAt: string
}

export interface CursorPage<T> {
  items: T[]
  nextCursor: string
}

export interface Signal {
  id: string
  chatTitle: string
  fromName: string
  contact: string
  text: string
  date: string
  score?: number
  isLead?: boolean
  semanticDirection?: string
  semanticCategory?: 'leads' | 'traders' | 'merchants' | 'ps_offers' | 'noise' | string
  categoryAssignedAt?: string
}

export interface LeadBrief {
  lead: Lead
  signals: Signal[]
  signalsCount: number
  lastSeenAt: string
}

export interface SignalItem {
  id: string
  chatTitle: string
  fromName: string
  contact: string
  text: string
  date: string
  leadId?: string | null
  leadScore?: number | null
  similarityScore?: number | null
  classifiedAsLead?: boolean | null
  semanticDirection?: string | null
  semanticCategory?: 'leads' | 'traders' | 'merchants' | 'ps_offers' | 'noise' | string
  classificationReason?: string | null
  traderScore?: number | null
  merchantScore?: number | null
  processingRequestScore?: number | null
  psOfferScore?: number | null
  noiseScore?: number | null
  primaryLabel?: string | null
  primaryPercent?: number | null
  categoryAssignedAt?: string
  senderTelegramId: number
  isIgnored: boolean
  isTeamMember: boolean
  isSpamSender?: boolean
  isDm: boolean
  otherChatsCount: number
  semanticFlags?: string[]
}

export interface ScoreBucket {
  from: number
  to: number
  count: number
  approved: number
  rejected: number
}

export interface LeadStats {
  period: string
  totalDetected: number
  approved: number
  rejected: number
  pending: number
  aiQualified: number
  manualApproved: number
  avgScore: number
  avgScoreApproved: number
  avgScoreRejected: number
  buckets: ScoreBucket[]
  approvedByCategory: {
    traders: number
    merchants: number
    psOffers: number
    other: number
  }
  rejectedByCategory: {
    traders: number
    merchants: number
    psOffers: number
    other: number
  }
  series: Array<{
    day: string
    traders: number
    merchants: number
    psOffers: number
    other: number
  }>
}

export interface IngestHourlyBucket {
  hour: string
  count: number
}

export interface IngestTopChat {
  chatId: number
  chatTitle: string
  count: number
}

export interface IngestStats {
  period: string
  totalSignals: number
  signalsToday: number
  signalsLastHour: number
  avgPerHour: number
  uniqueChats: number
  uniqueSenders: number
  leadCandidates: number
  teamMessages: number
  ignoredMessages: number
  lastSignalAt: string | null
  hourly: IngestHourlyBucket[]
  topChats: IngestTopChat[]
}

export interface Member {
  id: string
  name: string
  email: string
  tenant_id: string
  roles: BackendRole[]
  is_active: boolean
  created_at: string
  updated_at: string
  last_login?: string | null
}

export interface AppSettings {
  lead_threshold: string
  sender_window_seconds: string
  ignore_keywords?: string
  trader_threshold?: string
  merchant_threshold?: string
  ps_offer_threshold?: string
  noise_cleanup_enabled?: string
  show_multi_account_badges?: string
  tg_app_id?: string
  tg_app_hash?: string
  companies_json?: string
  [key: string]: string | undefined
}

export interface KnowledgeImportResult {
  fileName: string
  imported: number
  merchants: number
  psOffers: number
  traders: number
  noise: number
}

export interface Stat {
  title: string
  icon: string
  value: number | string
  variation: number
  formatter?: (value: number) => string
}

export interface Sale {
  id: string
  date: string
  status: SaleStatus
  email: string
  amount: number
}

export interface Notification {
  id: number
  unread?: boolean
  sender: User
  body: string
  date: string
}

export interface ChartDayBucket {
  day: string
  total: number
  target: number
  traders: number
  merchants: number
  psOffers: number
}

export type Period = 'daily' | 'weekly' | 'monthly'

export interface Range {
  start: Date
  end: Date
}
