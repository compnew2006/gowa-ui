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
import { MsgKey } from '../../whatsapp';
/**
 * Options for accepting an order
 */
export interface AcceptOrderOptions {
    /** The message ID or message key of the order message */
    msgId: string | MsgKey;
}
/**
 * Accept an order
 *
 * This function performs TWO critical operations:
 * 1. Updates the order message's status field from INQUIRY (1) to ACCEPTED (2)
 *    - This hides the "Accept Order" button in the UI
 * 2. Sends an order status update message to the customer
 *    - Shows "Processing" or "Confirmed" status
 *
 * The order information is automatically retrieved from the message.
 *
 * @example
 * ```javascript
 * // Accept an order with default "Processing" status
 * await WPP.order.accept({
 *   msgId: 'message_id_here',
 *   acceptNote: 'Your order has been confirmed and is being processed'
 * });
 *
 * // Accept with "Confirmed" status
 * await WPP.order.accept({
 *   msgId: 'message_id_here',
 *   orderStatus: WPP.order.OrderStatus.Confirmed,
 *   acceptNote: 'Your order has been confirmed'
 * });
 *
 * // Accept without a note
 * await WPP.order.accept({
 *   msgId: 'message_id_here'
 * });
 * ```
 *
 * @category Order
 */
export declare function accept(options: AcceptOrderOptions): Promise<void>;
