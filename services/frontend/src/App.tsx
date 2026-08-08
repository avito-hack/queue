import { Navigate, Route, Routes } from 'react-router-dom'
import { Header } from './components/layout/Header'
import { ToastHost } from './components/ToastHost'
import { useQueuePolling } from './features/queue/useQueuePolling'
import { AuthGate } from './features/session/AuthGate'
import { useBootstrapUserState } from './features/session/useBootstrapUserState'
import { useExpiredTickets } from './features/ticket/useExpiredTickets'
import { useTicketPolling } from './features/ticket/useTicketPolling'
import { Catalog } from './pages/catalog/Catalog'
import { Checkout } from './pages/checkout/Checkout'
import { Product } from './pages/product/Product'
import { Queue } from './pages/queue/Queue'

function AppReady() {
  useBootstrapUserState()
  useQueuePolling()
  useTicketPolling()
  useExpiredTickets()

  return (
    <div className="min-h-screen">
      <Header />
      <main className="mx-auto w-[min(1180px,calc(100%-22px))] px-0 pt-[18px] pb-20 sm:w-[min(1180px,calc(100%-32px))] sm:pt-7">
        <Routes>
          <Route path="/" element={<Navigate to="/catalog" replace />} />
          <Route path="/catalog" element={<Catalog />} />
          <Route path="/product/:id" element={<Product />} />
          <Route path="/queue" element={<Queue />} />
          <Route path="/checkout" element={<Checkout />} />
        </Routes>
      </main>
      <ToastHost />
    </div>
  )
}

function App() {
  return (
    <AuthGate>
      <AppReady />
    </AuthGate>
  )
}

export default App
