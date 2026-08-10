/** Статусы участника очереди — ItemQueueMemberStatus из OpenAPI. */
export type ItemQueueMemberStatus =
  | 'waiting_in_line'
  | 'acquired_purchase_rights'
  | 'placed_an_order'
  | 'purchased_an_item'
  | 'voluntarily_left_the_line'
  | 'given_up_purchase_rights'
  | 'lost_purchase_rights'
  | 'item_out_of_stock'

/** Состояние очереди товара — ItemQueueState из OpenAPI. */
export type ItemQueueState =
  | 'tickets_available'
  | 'tickets_partially_issued'
  | 'tickets_exhausted'

export type QueueStatus = 'queued' | 'soldout'

export type QueueEntry = {
  id: string
  productId: string
  status: QueueStatus
  position?: number
  /** Сырой статус с бэка, если нужен для отладки/расширений. */
  memberStatus?: ItemQueueMemberStatus
}
