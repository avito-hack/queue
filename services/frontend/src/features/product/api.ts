import { api } from "../../shared/api/client"

const getProducts = async () => {
    const response = await api.get(`/products`)
    return response.data
}

export const productApi = {
    getProducts,
}