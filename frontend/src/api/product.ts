import request from '../utils/request'
import type { PageResult, Product, ProductDetail } from '../types'

export interface ProductQuery {
  page?: number
  page_size?: number
  category?: string
  campus?: string
  keyword?: string
  status?: string
}

export function listProducts(params: ProductQuery) {
  return request.get<never, { code: number; message: string; data: PageResult<Product> }>('/products', { params })
}

export function getProduct(id: number) {
  return request.get<never, { code: number; message: string; data: ProductDetail }>(`/products/${id}`)
}

export interface CreateProductPayload {
  title: string
  description?: string
  price: number
  category: string
  condition: string
  campus: string
  trade_location: string
  images?: string
  slots?: { start_time: string }[]
}

export function createProduct(data: CreateProductPayload) {
  return request.post<never, { code: number; message: string; data: Product }>('/products', data)
}

export function removeProduct(id: number) {
  return request.delete<never, { code: number; message: string; data: Product }>(`/products/${id}`)
}

export function listGraduation() {
  return request.get<never, { code: number; message: string; data: PageResult<Product> }>('/products/graduation')
}
