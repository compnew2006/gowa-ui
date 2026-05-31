/*!
 * Copyright 2023 WPPConnect Team
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
import { WPPError } from '../../util';
import { Wid } from '../../whatsapp';
export interface PnLidWid {
    id: string;
    server: string;
    _serialized: string;
}
export interface PnLidContactInfo {
    name?: string;
    shortName?: string;
    pushname?: string;
    type?: 'in' | 'out';
    verifiedName?: string;
    isBusiness?: boolean;
    isEnterprise?: boolean;
    verifiedLevel?: number;
    syncToAddressbook?: boolean;
    isContactSyncCompleted?: number;
}
export interface PnLidEntryResult {
    lid?: PnLidWid;
    phoneNumber?: PnLidWid;
    contact?: PnLidContactInfo;
}
export declare class InvalidWidForGetPnLidEntry extends WPPError {
    readonly id: string | {
        _serialized: string;
    };
    constructor(id: string | {
        _serialized: string;
    });
}
/**
 * Get LID/PhoneNumber mapping and Contact information
 *
 * @example
 * ```javascript
 * await WPP.contact.getPnLidEntry('[number]@c.us');
 * await WPP.contact.getPnLidEntry('[number]@lid');
 * ```
 *
 * @category Contact
 */
export declare function getPnLidEntry(contactId: string | Wid): Promise<PnLidEntryResult>;
