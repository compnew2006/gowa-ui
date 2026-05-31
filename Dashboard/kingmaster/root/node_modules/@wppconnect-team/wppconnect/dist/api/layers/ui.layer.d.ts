import { AutoDownloadSettings, Theme } from '@wppconnect/wa-js/dist/conn';
import { Page } from 'puppeteer';
import { CreateConfig } from '../../config/create-config';
import { Wid } from '../model';
import { GroupLayer } from './group.layer';
export declare class UILayer extends GroupLayer {
    page: Page;
    constructor(page: Page, session?: string, options?: CreateConfig);
    /**
     * Opens given chat at last message (bottom)
     * Will fire natural workflow events of whatsapp web
     * @category UI
     * @param chatId
     */
    openChat(chatId: string | Wid): Promise<boolean>;
    /**
     * Opens chat at given message position
     * @category UI
     * @param chatId Chat id
     * @param messageId Message id (For example: '06D3AB3D0EEB9D077A3F9A3EFF4DD030')
     */
    openChatAt(chatId: string | Wid, messageId: string): Promise<boolean>;
    /**
     * Closes the currently opened chat (if any).
     * The boolean result reflects if there was any chat that got closed.
     * @category UI
     */
    closeChat(): Promise<boolean>;
    /**
     * Return the currently active chat (visually open)
     * @category UI
     */
    getActiveChat(): Promise<import("@wppconnect/wa-js/dist/whatsapp").ChatModel>;
    /**
     * Get current theme
     * @category UI
     * @returns Current theme ('dark' or 'light')
     */
    getTheme(): Promise<Theme>;
    /**
     * Set theme
     * Note: This will force a reload of WhatsApp Web to take effect
     * @category UI
     * @param theme Theme to set ('dark' or 'light')
     * @returns void
     */
    setTheme(theme: Theme): Promise<void>;
    /**
     * Get auto download settings
     * @category UI
     * @returns Auto download settings
     */
    getAutoDownloadSettings(): Promise<{
        photos: boolean;
        audio: boolean;
        videos: boolean;
        documents: boolean;
    }>;
    /**
     * Set auto download settings
     * Note: This will force a reload of WhatsApp Web to take effect
     * @category UI
     * @param settings Auto download settings to set
     * @returns void
     */
    setAutoDownloadSettings(settings: AutoDownloadSettings): Promise<void>;
}
