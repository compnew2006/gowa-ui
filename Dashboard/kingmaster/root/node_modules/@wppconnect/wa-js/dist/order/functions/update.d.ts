/*!
 * Copyright 2025 WPPConnect Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { UpdateOrderOptions } from '../types';
/**
 * Update an order status and/or payment information
 *
 * This function sends an order status update message to the customer.
 *
 * **Important:** For accepting or declining orders, use the dedicated `accept()` and `decline()`
 * functions instead. Those functions properly update both the message status field (UI state)
 * and send the status update message.
 *
 * This function is best used for status updates after an order has been accepted, such as:
 * - Marking as shipped
 * - Marking as delivered
 * - Marking as complete
 * - Updating payment information
 *
 * @example
 * ```javascript
 * // Mark as shipped
 * await WPP.order.update({
 *   msgId: 'message_id_here',
 *   orderInfo: {
 *     items: [
 *       {
 *         id: 'product_1',
 *         name: 'Product Name',
 *         amount: 1000,
 *         quantity: 2,
 *         isCustomItem: false,
 *         isQuantitySet: true
 *       }
 *     ],
 *     totalAmount: 2000,
 *     subtotal: 2000,
 *     currency: 'USD'
 *   },
 *   orderStatus: WPP.order.OrderStatus.Shipped,
 *   orderNote: 'Your order has been shipped',
 *   referenceId: 'ORDER_12345'
 * });
 *
 * // Mark as delivered
 * await WPP.order.update({
 *   msgId: 'message_id_here',
 *   orderInfo: { ... },
 *   orderStatus: WPP.order.OrderStatus.Delivered,
 *   orderNote: 'Your order has been delivered'
 * });
 *
 * // Update payment status
 * await WPP.order.update({
 *   msgId: 'message_id_here',
 *   orderInfo: { ... },
 *   orderStatus: WPP.order.OrderStatus.Complete,
 *   paymentStatus: WPP.order.PaymentStatus.Captured,
 *   orderNote: 'Order completed and payment received'
 * });
 * ```
 *
 * @category Order
 */
export declare function update(options: UpdateOrderOptions): Promise<void>;
