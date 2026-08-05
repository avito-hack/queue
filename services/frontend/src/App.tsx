import { Navigate, Route, Routes } from 'react-router-dom'
import { Header } from './components/layout/Header'
import { Catalog } from './pages/catalog/Catalog'
import { Product } from './pages/product/Product'
import { Queue } from './pages/queue/Queue'

function App() {
  return (
    <div className="min-h-screen">
      <Header />
      <main className="mx-auto w-[min(1180px,calc(100%-22px))] px-0 pt-[18px] pb-20 sm:w-[min(1180px,calc(100%-32px))] sm:pt-7">
        <Routes>
          <Route path="/" element={<Navigate to="/catalog" replace />} />
          <Route path="/catalog" element={<Catalog />} />
          <Route path="/product/:id" element={<Product />} />
          <Route path="/queue" element={<Queue />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
