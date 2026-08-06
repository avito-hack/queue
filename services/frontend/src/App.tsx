import { Navigate, Route, Routes } from 'react-router-dom'
import { Header } from './components/layout/Header'
import { useQueuePolling } from './features/queue/useQueuePolling'
import { useExpiredTickets } from './features/ticket/useExpiredTickets'
import { useTicketPolling } from './features/ticket/useTicketPolling'
import { Catalog } from './pages/catalog/Catalog'
import { Checkout } from './pages/checkout/Checkout'
import { Product } from './pages/product/Product'
import { Queue } from './pages/queue/Queue'

function App() {
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
    </div>
  )
}

export default App
