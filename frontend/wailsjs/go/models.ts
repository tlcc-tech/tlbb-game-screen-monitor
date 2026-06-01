export namespace main {
	
	export class AppInfo {
	    name: string;
	    author: string;
	    version: string;
	    repoUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.author = source["author"];
	        this.version = source["version"];
	        this.repoUrl = source["repoUrl"];
	    }
	}
	export class TemplateItem {
	    id: string;
	    name: string;
	    file: string;
	    threshold: number;
	    enabled: boolean;
	    category: string;
	    presetKey: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplateItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.file = source["file"];
	        this.threshold = source["threshold"];
	        this.enabled = source["enabled"];
	        this.category = source["category"];
	        this.presetKey = source["presetKey"];
	    }
	}
	export class AppSettings {
	    channelKey: string;
	    pollIntervalSec: number;
	    consecutiveHits: number;
	    pushCooldownMin: number;
	    networkWaitMaxMin: number;
	    pingHost: string;
	    httpProbeUrl: string;
	    usePing: boolean;
	    useHttp: boolean;
	    gameWindowTitle: string;
	    gameWindowHwnd: number;
	    hotkeyStart: string;
	    hotkeyStop: string;
	    notifyOnRecover: boolean;
	    templates: TemplateItem[];
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.channelKey = source["channelKey"];
	        this.pollIntervalSec = source["pollIntervalSec"];
	        this.consecutiveHits = source["consecutiveHits"];
	        this.pushCooldownMin = source["pushCooldownMin"];
	        this.networkWaitMaxMin = source["networkWaitMaxMin"];
	        this.pingHost = source["pingHost"];
	        this.httpProbeUrl = source["httpProbeUrl"];
	        this.usePing = source["usePing"];
	        this.useHttp = source["useHttp"];
	        this.gameWindowTitle = source["gameWindowTitle"];
	        this.gameWindowHwnd = source["gameWindowHwnd"];
	        this.hotkeyStart = source["hotkeyStart"];
	        this.hotkeyStop = source["hotkeyStop"];
	        this.notifyOnRecover = source["notifyOnRecover"];
	        this.templates = this.convertValues(source["templates"], TemplateItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MatchScore {
	    templateId: string;
	    templateName: string;
	    category: string;
	    presetKey: string;
	    score: number;
	    found: boolean;
	    matched: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MatchScore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.templateId = source["templateId"];
	        this.templateName = source["templateName"];
	        this.category = source["category"];
	        this.presetKey = source["presetKey"];
	        this.score = source["score"];
	        this.found = source["found"];
	        this.matched = source["matched"];
	    }
	}
	export class TemplatePushCooldown {
	    templateId: string;
	    templateName: string;
	    remainingSec: number;
	
	    static createFrom(source: any = {}) {
	        return new TemplatePushCooldown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.templateId = source["templateId"];
	        this.templateName = source["templateName"];
	        this.remainingSec = source["remainingSec"];
	    }
	}
	export class MonitorStatus {
	    running: boolean;
	    phase: string;
	    lastScores: MatchScore[];
	    lastMatchedName: string;
	    lastMatchedScore: number;
	    lastChecked: string;
	    lastError: string;
	    nextPollIn: number;
	    pendingPush: boolean;
	    pendingTemplates: string[];
	    pushCooldowns: TemplatePushCooldown[];
	
	    static createFrom(source: any = {}) {
	        return new MonitorStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.phase = source["phase"];
	        this.lastScores = this.convertValues(source["lastScores"], MatchScore);
	        this.lastMatchedName = source["lastMatchedName"];
	        this.lastMatchedScore = source["lastMatchedScore"];
	        this.lastChecked = source["lastChecked"];
	        this.lastError = source["lastError"];
	        this.nextPollIn = source["nextPollIn"];
	        this.pendingPush = source["pendingPush"];
	        this.pendingTemplates = source["pendingTemplates"];
	        this.pushCooldowns = this.convertValues(source["pushCooldowns"], TemplatePushCooldown);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class WindowInfo {
	    hwnd: number;
	    title: string;
	    className: string;
	
	    static createFrom(source: any = {}) {
	        return new WindowInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hwnd = source["hwnd"];
	        this.title = source["title"];
	        this.className = source["className"];
	    }
	}

}

