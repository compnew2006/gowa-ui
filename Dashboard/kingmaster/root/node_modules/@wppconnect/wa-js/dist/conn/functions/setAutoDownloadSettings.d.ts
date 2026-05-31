/*!
 * Copyright 2026 WPPConnect Team
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
export interface AutoDownloadSettings {
    /**
     * Enable/disable auto-download for photos
     */
    photos?: boolean;
    /**
     * Enable/disable auto-download for audio
     */
    audio?: boolean;
    /**
     * Enable/disable auto-download for videos
     */
    videos?: boolean;
    /**
     * Enable/disable auto-download for documents
     */
    documents?: boolean;
}
/**
 * Set auto-download settings for media types
 *
 * @example
 * ```javascript
 * // Disable video and document auto-download
 * await WPP.conn.setAutoDownloadSettings({
 *   videos: false,
 *   documents: false
 * });
 *
 * // Enable only photos
 * await WPP.conn.setAutoDownloadSettings({
 *   photos: true,
 *   audio: false,
 *   videos: false,
 *   documents: false
 * });
 * ```
 *
 * @category Config
 */
export declare function setAutoDownloadSettings(settings: AutoDownloadSettings): Promise<void>;
