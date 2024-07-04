import { inspect } from "node:util"

/**
 * Enum for log levels
 * @readonly
 * @enum {number}
 */
export const Levels = {
    ERROR: 1,
    WARN: 2,
    INFO: 3,
    DEBUG: 4,
}


export class Logger {
    /** @type {string} */
    #prefix;
    #level;

    constructor(prefix, level = Levels.INFO) {
        this.#prefix = prefix;
        this.#level = level;
    }

    /**
     * @param {string} message
     * @param {string} level
     */
    #log = (message, level) => {
        if (Levels[level] > this.#level) {
            return;
        }

        const writer = {
            ERROR: console.error,
            WARN: console.warn,
            INFO: console.info,
            DEBUG: console.debug,
        }[level] ?? console.log;

        const prefixName = this.#prefix || "Brunodo";
        const prefix = `[${prefixName} ${level}]`;
        const msg = inspect(message, { colors: true, depth: 4 })

        writer(prefix, msg);
    }

    error = (message) => {
        this.#log(message, "ERROR");
    }

    warn = (message) => {
        this.#log(message, "WARN");
    }

    info = (message) => {
        this.#log(message, "INFO");
    }

    debug = (message) => {
        this.#log(message, "DEBUG");
    }
}