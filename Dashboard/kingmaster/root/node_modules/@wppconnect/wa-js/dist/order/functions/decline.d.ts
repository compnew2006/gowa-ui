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
 * Options for declining an order
 */
export interface DeclineOrderOptions {
    /** The message ID or message key of the order message */
    msgId: string | MsgKey;
    /** Optional note to include with the decline message (recommended) */
    declineNote?: string;
}
/**
 * Decline an order
 *
 * This function performs TWO critical operations:
 * 1. Updates the order message's status field from INQUIRY (1) to DECLINED (3)
 *    - This changes the UI to show the order as declined
 * 2. Sends an order cancellation message to the customer
 *    - Shows "Canceled" status with optional note
 *
 * The order information is automatically retrieved from the message.
 *
 * @example
 * ```javascript
 * // Decline an order with a note
 * await WPP.order.decline({
 *   msgId: 'message_id_here',
 *   declineNote: 'Sorry, we cannot fulfill this order at this time'
 * });
 *
 * // Decline without a note
 * await WPP.order.decline({
 *   msgId: 'message_id_here'
 * });
 * ```
 *
 * @category Order
 */
export declare function decline(options: DeclineOrderOptions): Promise<void>;
