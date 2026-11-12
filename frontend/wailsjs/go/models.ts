export namespace app {
	
	export class AtlasKeyStatus {
	    set: boolean;
	    hint: string;
	
	    static createFrom(source: any = {}) {
	        return new AtlasKeyStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.set = source["set"];
	        this.hint = source["hint"];
	    }
	}
	export class IPSpaceState {
	    exists: boolean;
	    unlocked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new IPSpaceState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exists = source["exists"];
	        this.unlocked = source["unlocked"];
	    }
	}
	export class ImportInfo {
	    opened: boolean;
	    name: string;
	    mode: string;
	    count: number;
	    hasSync: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImportInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.opened = source["opened"];
	        this.name = source["name"];
	        this.mode = source["mode"];
	        this.count = source["count"];
	        this.hasSync = source["hasSync"];
	    }
	}
	export class InvInfo {
	    id: string;
	    name: string;
	    active: boolean;
	    unlocked: boolean;
	    hasSync: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InvInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.active = source["active"];
	        this.unlocked = source["unlocked"];
	        this.hasSync = source["hasSync"];
	    }
	}
	export class NetflowView {
	    running: boolean;
	    port: number;
	    flows: number;
	    exporters: number;
	    windowSecCovered: number;
	    truncated: boolean;
	    top: netflow.Conversation[];
	
	    static createFrom(source: any = {}) {
	        return new NetflowView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.flows = source["flows"];
	        this.exporters = source["exporters"];
	        this.windowSecCovered = source["windowSecCovered"];
	        this.truncated = source["truncated"];
	        this.top = this.convertValues(source["top"], netflow.Conversation);
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
	export class PullReport {
	    added: number;
	    skipped: string[];
	
	    static createFrom(source: any = {}) {
	        return new PullReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.skipped = source["skipped"];
	    }
	}
	export class SyncInfo {
	    configured: boolean;
	    kind: string;
	    location: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configured = source["configured"];
	        this.kind = source["kind"];
	        this.location = source["location"];
	    }
	}
	export class SyncResult {
	    incoming: ipspace.Change[];
	    outgoing: ipspace.Change[];
	    conflicts: ipspace.Conflict[];
	    duplicatePrefixes: ipspace.PrefixCollision[];
	    applied: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.incoming = this.convertValues(source["incoming"], ipspace.Change);
	        this.outgoing = this.convertValues(source["outgoing"], ipspace.Change);
	        this.conflicts = this.convertValues(source["conflicts"], ipspace.Conflict);
	        this.duplicatePrefixes = this.convertValues(source["duplicatePrefixes"], ipspace.PrefixCollision);
	        this.applied = source["applied"];
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
	export class TftpView {
	    running: boolean;
	    port: number;
	    files: tftp.File[];
	
	    static createFrom(source: any = {}) {
	        return new TftpView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.files = this.convertValues(source["files"], tftp.File);
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

}

export namespace atlas {
	
	export class Hop {
	    number: number;
	    from: string;
	    rttMs: number;
	    timeout: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Hop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.from = source["from"];
	        this.rttMs = source["rttMs"];
	        this.timeout = source["timeout"];
	    }
	}
	export class Probe {
	    id: number;
	    from: string;
	    minMs: number;
	    avgMs: number;
	    maxMs: number;
	    sent: number;
	    received: number;
	
	    static createFrom(source: any = {}) {
	        return new Probe(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.from = source["from"];
	        this.minMs = source["minMs"];
	        this.avgMs = source["avgMs"];
	        this.maxMs = source["maxMs"];
	        this.sent = source["sent"];
	        this.received = source["received"];
	    }
	}
	export class Report {
	    target: string;
	    measurementId: number;
	    probes: Probe[];
	    responding: number;
	    minMs: number;
	    avgMs: number;
	    maxMs: number;
	    lossPct: number;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.measurementId = source["measurementId"];
	        this.probes = this.convertValues(source["probes"], Probe);
	        this.responding = source["responding"];
	        this.minMs = source["minMs"];
	        this.avgMs = source["avgMs"];
	        this.maxMs = source["maxMs"];
	        this.lossPct = source["lossPct"];
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
	export class TraceProbe {
	    id: number;
	    from: string;
	    dest: string;
	    hops: Hop[];
	    reached: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TraceProbe(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.from = source["from"];
	        this.dest = source["dest"];
	        this.hops = this.convertValues(source["hops"], Hop);
	        this.reached = source["reached"];
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
	export class TraceReport {
	    target: string;
	    measurementId: number;
	    requested: number;
	    probes: TraceProbe[];
	    responding: number;
	    reached: number;
	
	    static createFrom(source: any = {}) {
	        return new TraceReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.measurementId = source["measurementId"];
	        this.requested = source["requested"];
	        this.probes = this.convertValues(source["probes"], TraceProbe);
	        this.responding = source["responding"];
	        this.reached = source["reached"];
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

}

export namespace ciscopw {
	
	export class Secret {
	    context: string;
	    keyword: string;
	    type: string;
	    encoded: string;
	    plaintext: string;
	    decoded: boolean;
	    verdict: string;
	
	    static createFrom(source: any = {}) {
	        return new Secret(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.context = source["context"];
	        this.keyword = source["keyword"];
	        this.type = source["type"];
	        this.encoded = source["encoded"];
	        this.plaintext = source["plaintext"];
	        this.decoded = source["decoded"];
	        this.verdict = source["verdict"];
	    }
	}

}

export namespace collector {
	
	export class SyslogMsg {
	    // Go type: time
	    t: any;
	    source: string;
	    facility: number;
	    severity: number;
	    severityName: string;
	    tag: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SyslogMsg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.t = this.convertValues(source["t"], null);
	        this.source = source["source"];
	        this.facility = source["facility"];
	        this.severity = source["severity"];
	        this.severityName = source["severityName"];
	        this.tag = source["tag"];
	        this.message = source["message"];
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
	export class Stamped_github_com_hriday_meridian_internal_collector_SyslogMsg_ {
	    seq: number;
	    val: SyslogMsg;
	
	    static createFrom(source: any = {}) {
	        return new Stamped_github_com_hriday_meridian_internal_collector_SyslogMsg_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.val = this.convertValues(source["val"], SyslogMsg);
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
	export class Poll_github_com_hriday_meridian_internal_collector_SyslogMsg_ {
	    running: boolean;
	    port: number;
	    epoch: number;
	    total: number;
	    dropped: number;
	    items: Stamped_github_com_hriday_meridian_internal_collector_SyslogMsg_[];
	
	    static createFrom(source: any = {}) {
	        return new Poll_github_com_hriday_meridian_internal_collector_SyslogMsg_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.epoch = source["epoch"];
	        this.total = source["total"];
	        this.dropped = source["dropped"];
	        this.items = this.convertValues(source["items"], Stamped_github_com_hriday_meridian_internal_collector_SyslogMsg_);
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
	export class Stamped_github_com_hriday_meridian_internal_snmp_TrapMsg_ {
	    seq: number;
	    val: snmp.TrapMsg;
	
	    static createFrom(source: any = {}) {
	        return new Stamped_github_com_hriday_meridian_internal_snmp_TrapMsg_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.val = this.convertValues(source["val"], snmp.TrapMsg);
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
	export class Poll_github_com_hriday_meridian_internal_snmp_TrapMsg_ {
	    running: boolean;
	    port: number;
	    epoch: number;
	    total: number;
	    dropped: number;
	    items: Stamped_github_com_hriday_meridian_internal_snmp_TrapMsg_[];
	
	    static createFrom(source: any = {}) {
	        return new Poll_github_com_hriday_meridian_internal_snmp_TrapMsg_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.epoch = source["epoch"];
	        this.total = source["total"];
	        this.dropped = source["dropped"];
	        this.items = this.convertValues(source["items"], Stamped_github_com_hriday_meridian_internal_snmp_TrapMsg_);
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
	
	

}

export namespace configdiff {
	
	export class Line {
	    op: string;
	    text: string;
	    section: string;
	
	    static createFrom(source: any = {}) {
	        return new Line(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.op = source["op"];
	        this.text = source["text"];
	        this.section = source["section"];
	    }
	}
	export class Result {
	    lines: Line[];
	    added: number;
	    removed: number;
	    sections: string[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = this.convertValues(source["lines"], Line);
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.sections = source["sections"];
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

}

export namespace dnsaudit {
	
	export class Check {
	    name: string;
	    status: string;
	    note: string;
	    records?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Check(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.note = source["note"];
	        this.records = source["records"];
	    }
	}
	export class Report {
	    domain: string;
	    resolver: string;
	    checks: Check[];
	    oks: number;
	    warns: number;
	    missing: number;
	    errors: number;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.resolver = source["resolver"];
	        this.checks = this.convertValues(source["checks"], Check);
	        this.oks = source["oks"];
	        this.warns = source["warns"];
	        this.missing = source["missing"];
	        this.errors = source["errors"];
	        this.summary = source["summary"];
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

}

export namespace dnsbl {
	
	export class Listing {
	    zone: string;
	    name: string;
	    status: string;
	    codes?: string[];
	    reason?: string;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new Listing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.zone = source["zone"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.codes = source["codes"];
	        this.reason = source["reason"];
	        this.note = source["note"];
	    }
	}
	export class Report {
	    ip: string;
	    resolver: string;
	    checked: number;
	    listed: number;
	    unreachable: number;
	    verdict: string;
	    summary: string;
	    listings: Listing[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.resolver = source["resolver"];
	        this.checked = source["checked"];
	        this.listed = source["listed"];
	        this.unreachable = source["unreachable"];
	        this.verdict = source["verdict"];
	        this.summary = source["summary"];
	        this.listings = this.convertValues(source["listings"], Listing);
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

}

export namespace dnsx {
	
	export class Record {
	    name: string;
	    type: string;
	    ttl: number;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.ttl = source["ttl"];
	        this.value = source["value"];
	    }
	}
	export class Result {
	    server: string;
	    rcode: string;
	    flags: string[];
	    authenticated: boolean;
	    rttMillis: number;
	    answers: Record[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.rcode = source["rcode"];
	        this.flags = source["flags"];
	        this.authenticated = source["authenticated"];
	        this.rttMillis = source["rttMillis"];
	        this.answers = this.convertValues(source["answers"], Record);
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

}

export namespace fwconfig {
	
	export class Address {
	    name: string;
	    kind: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new Address(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.value = source["value"];
	    }
	}
	export class Counts {
	    vlans: number;
	    policies: number;
	    addresses: number;
	    subnetsIps: number;
	    services: number;
	    zones: number;
	
	    static createFrom(source: any = {}) {
	        return new Counts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vlans = source["vlans"];
	        this.policies = source["policies"];
	        this.addresses = source["addresses"];
	        this.subnetsIps = source["subnetsIps"];
	        this.services = source["services"];
	        this.zones = source["zones"];
	    }
	}
	export class Flow {
	    src: string;
	    dst: string;
	    services: string[];
	    action: string;
	    policyIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new Flow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.src = source["src"];
	        this.dst = source["dst"];
	        this.services = source["services"];
	        this.action = source["action"];
	        this.policyIds = source["policyIds"];
	    }
	}
	export class Policy {
	    id: number;
	    name: string;
	    srcIntf: string[];
	    dstIntf: string[];
	    srcAddr: string[];
	    dstAddr: string[];
	    services: string[];
	    action: string;
	    disabled: boolean;
	    negated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Policy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.srcIntf = source["srcIntf"];
	        this.dstIntf = source["dstIntf"];
	        this.srcAddr = source["srcAddr"];
	        this.dstAddr = source["dstAddr"];
	        this.services = source["services"];
	        this.action = source["action"];
	        this.disabled = source["disabled"];
	        this.negated = source["negated"];
	    }
	}
	export class Service {
	    name: string;
	    proto: string;
	    ports: string;
	
	    static createFrom(source: any = {}) {
	        return new Service(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.proto = source["proto"];
	        this.ports = source["ports"];
	    }
	}
	export class Zone {
	    name: string;
	    interfaces: string[];
	
	    static createFrom(source: any = {}) {
	        return new Zone(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.interfaces = source["interfaces"];
	    }
	}
	export class VLAN {
	    id: number;
	    name: string;
	    interface: string;
	    subnet: string;
	    zone: string;
	
	    static createFrom(source: any = {}) {
	        return new VLAN(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.interface = source["interface"];
	        this.subnet = source["subnet"];
	        this.zone = source["zone"];
	    }
	}
	export class Analysis {
	    vdom: string;
	    vlans: VLAN[];
	    zones: Zone[];
	    addresses: Address[];
	    services: Service[];
	    policies: Policy[];
	    flows: Flow[];
	    counts: Counts;
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new Analysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vdom = source["vdom"];
	        this.vlans = this.convertValues(source["vlans"], VLAN);
	        this.zones = this.convertValues(source["zones"], Zone);
	        this.addresses = this.convertValues(source["addresses"], Address);
	        this.services = this.convertValues(source["services"], Service);
	        this.policies = this.convertValues(source["policies"], Policy);
	        this.flows = this.convertValues(source["flows"], Flow);
	        this.counts = this.convertValues(source["counts"], Counts);
	        this.notes = source["notes"];
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
	
	
	
	
	

}

export namespace httpcheck {
	
	export class Hop {
	    status: number;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new Hop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.url = source["url"];
	    }
	}
	export class Timing {
	    dnsMillis: number;
	    connectMillis: number;
	    tlsMillis: number;
	    ttfbMillis: number;
	    totalMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new Timing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dnsMillis = source["dnsMillis"];
	        this.connectMillis = source["connectMillis"];
	        this.tlsMillis = source["tlsMillis"];
	        this.ttfbMillis = source["ttfbMillis"];
	        this.totalMillis = source["totalMillis"];
	    }
	}
	export class Result {
	    url: string;
	    finalUrl: string;
	    status: number;
	    statusText: string;
	    proto: string;
	    tlsVersion: string;
	    server: string;
	    contentType: string;
	    contentLength: number;
	    redirects: Hop[];
	    headers: string[][];
	    timing: Timing;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.finalUrl = source["finalUrl"];
	        this.status = source["status"];
	        this.statusText = source["statusText"];
	        this.proto = source["proto"];
	        this.tlsVersion = source["tlsVersion"];
	        this.server = source["server"];
	        this.contentType = source["contentType"];
	        this.contentLength = source["contentLength"];
	        this.redirects = this.convertValues(source["redirects"], Hop);
	        this.headers = source["headers"];
	        this.timing = this.convertValues(source["timing"], Timing);
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

}

export namespace ipspace {
	
	export class Record {
	    id: string;
	    prefix: string;
	    name: string;
	    status: string;
	    tags: string[];
	    notes: string;
	    parentId: string;
	    createdBy: string;
	    changedBy: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    changedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.prefix = source["prefix"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.tags = source["tags"];
	        this.notes = source["notes"];
	        this.parentId = source["parentId"];
	        this.createdBy = source["createdBy"];
	        this.changedBy = source["changedBy"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.changedAt = this.convertValues(source["changedAt"], null);
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
	export class Change {
	    kind: string;
	    record: Record;
	    prev?: Record;
	
	    static createFrom(source: any = {}) {
	        return new Change(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.record = this.convertValues(source["record"], Record);
	        this.prev = this.convertValues(source["prev"], Record);
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
	export class Conflict {
	    id: string;
	    prefix: string;
	    base: Record;
	    local: Record;
	    remote: Record;
	    localDeleted: boolean;
	    remoteDeleted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Conflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.prefix = source["prefix"];
	        this.base = this.convertValues(source["base"], Record);
	        this.local = this.convertValues(source["local"], Record);
	        this.remote = this.convertValues(source["remote"], Record);
	        this.localDeleted = source["localDeleted"];
	        this.remoteDeleted = source["remoteDeleted"];
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
	export class Event {
	    id: string;
	    type: string;
	    actor: string;
	    // Go type: time
	    at: any;
	    record: Record;
	
	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.actor = source["actor"];
	        this.at = this.convertValues(source["at"], null);
	        this.record = this.convertValues(source["record"], Record);
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
	export class ImportReport {
	    added: number;
	    skipped: number;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	    }
	}
	export class PrefixCollision {
	    prefix: string;
	    records: Record[];
	
	    static createFrom(source: any = {}) {
	        return new PrefixCollision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prefix = source["prefix"];
	        this.records = this.convertValues(source["records"], Record);
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
	
	export class TreeNode {
	    record: Record;
	    children: TreeNode[];
	
	    static createFrom(source: any = {}) {
	        return new TreeNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.record = this.convertValues(source["record"], Record);
	        this.children = this.convertValues(source["children"], TreeNode);
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

}

export namespace lookup {
	
	export class PeerFac {
	    name: string;
	    city: string;
	    country: string;
	
	    static createFrom(source: any = {}) {
	        return new PeerFac(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.city = source["city"];
	        this.country = source["country"];
	    }
	}
	export class PeerIX {
	    name: string;
	    speed: number;
	
	    static createFrom(source: any = {}) {
	        return new PeerIX(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.speed = source["speed"];
	    }
	}
	export class Peering {
	    name: string;
	    type: string;
	    scope: string;
	    ratio: string;
	    policy: string;
	    irrAsSet: string;
	    website: string;
	    v4Prefixes: number;
	    v6Prefixes: number;
	    ixCount: number;
	    facCount: number;
	    ixps: PeerIX[];
	    facilities: PeerFac[];
	
	    static createFrom(source: any = {}) {
	        return new Peering(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.scope = source["scope"];
	        this.ratio = source["ratio"];
	        this.policy = source["policy"];
	        this.irrAsSet = source["irrAsSet"];
	        this.website = source["website"];
	        this.v4Prefixes = source["v4Prefixes"];
	        this.v6Prefixes = source["v6Prefixes"];
	        this.ixCount = source["ixCount"];
	        this.facCount = source["facCount"];
	        this.ixps = this.convertValues(source["ixps"], PeerIX);
	        this.facilities = this.convertValues(source["facilities"], PeerFac);
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
	export class Source {
	    label: string;
	    // Go type: time
	    fetchedAt: any;
	    err: string;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.fetchedAt = this.convertValues(source["fetchedAt"], null);
	        this.err = source["err"];
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
	export class ASNReport {
	    asn: string;
	    holder: string;
	    registry: string;
	    v4Prefixes: number;
	    v6Prefixes: number;
	    neighbours: number;
	    routingSrc: Source;
	    registrySrc: Source;
	    peering?: Peering;
	    peeringSrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new ASNReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.asn = source["asn"];
	        this.holder = source["holder"];
	        this.registry = source["registry"];
	        this.v4Prefixes = source["v4Prefixes"];
	        this.v6Prefixes = source["v6Prefixes"];
	        this.neighbours = source["neighbours"];
	        this.routingSrc = this.convertValues(source["routingSrc"], Source);
	        this.registrySrc = this.convertValues(source["registrySrc"], Source);
	        this.peering = this.convertValues(source["peering"], Peering);
	        this.peeringSrc = this.convertValues(source["peeringSrc"], Source);
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
	export class BGPCollector {
	    rrc: string;
	    location: string;
	    scope: string;
	    peers: number;
	
	    static createFrom(source: any = {}) {
	        return new BGPCollector(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rrc = source["rrc"];
	        this.location = source["location"];
	        this.scope = source["scope"];
	        this.peers = source["peers"];
	    }
	}
	export class BGPPath {
	    path: string[];
	    peers: number;
	    percent: number;
	
	    static createFrom(source: any = {}) {
	        return new BGPPath(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.peers = source["peers"];
	        this.percent = source["percent"];
	    }
	}
	export class BGPReport {
	    input: string;
	    prefix: string;
	    originAsns: string[];
	    moas: boolean;
	    collectors: number;
	    peers: number;
	    paths: BGPPath[];
	    upstreams: string[];
	    transit: string[];
	    rpkiState: string;
	    vantages: BGPCollector[];
	    routingSrc: Source;
	    rpkiSrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new BGPReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.prefix = source["prefix"];
	        this.originAsns = source["originAsns"];
	        this.moas = source["moas"];
	        this.collectors = source["collectors"];
	        this.peers = source["peers"];
	        this.paths = this.convertValues(source["paths"], BGPPath);
	        this.upstreams = source["upstreams"];
	        this.transit = source["transit"];
	        this.rpkiState = source["rpkiState"];
	        this.vantages = this.convertValues(source["vantages"], BGPCollector);
	        this.routingSrc = this.convertValues(source["routingSrc"], Source);
	        this.rpkiSrc = this.convertValues(source["rpkiSrc"], Source);
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
	export class DomainReport {
	    input: string;
	    rcode: string;
	    records: dnsx.Record[];
	    resolvesTo: string[];
	    hostedBy: string[];
	    registrar: string;
	    created: string;
	    updated: string;
	    expires: string;
	    dnssec: string;
	    statuses: string[];
	    nameservers: string[];
	    abuseEmail: string;
	    dnsSrc: Source;
	    registrySrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new DomainReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.rcode = source["rcode"];
	        this.records = this.convertValues(source["records"], dnsx.Record);
	        this.resolvesTo = source["resolvesTo"];
	        this.hostedBy = source["hostedBy"];
	        this.registrar = source["registrar"];
	        this.created = source["created"];
	        this.updated = source["updated"];
	        this.expires = source["expires"];
	        this.dnssec = source["dnssec"];
	        this.statuses = source["statuses"];
	        this.nameservers = source["nameservers"];
	        this.abuseEmail = source["abuseEmail"];
	        this.dnsSrc = this.convertValues(source["dnsSrc"], Source);
	        this.registrySrc = this.convertValues(source["registrySrc"], Source);
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
	export class IPReport {
	    input: string;
	    version: string;
	    coveringPrefix: string;
	    originAsn: string;
	    originName: string;
	    registeredOrg: string;
	    abuseEmail: string;
	    registry: string;
	    rpkiState: string;
	    routingNote: string;
	    routingSrc: Source;
	    registrySrc: Source;
	    peering?: Peering;
	    peeringSrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new IPReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.version = source["version"];
	        this.coveringPrefix = source["coveringPrefix"];
	        this.originAsn = source["originAsn"];
	        this.originName = source["originName"];
	        this.registeredOrg = source["registeredOrg"];
	        this.abuseEmail = source["abuseEmail"];
	        this.registry = source["registry"];
	        this.rpkiState = source["rpkiState"];
	        this.routingNote = source["routingNote"];
	        this.routingSrc = this.convertValues(source["routingSrc"], Source);
	        this.registrySrc = this.convertValues(source["registrySrc"], Source);
	        this.peering = this.convertValues(source["peering"], Peering);
	        this.peeringSrc = this.convertValues(source["peeringSrc"], Source);
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
	
	
	
	export class PrefixReport {
	    input: string;
	    originAsn: string;
	    originName: string;
	    announced: boolean;
	    registry: string;
	    registeredOrg: string;
	    abuseEmail: string;
	    rpkiState: string;
	    rpkiRoa: string;
	    lessSpecific: string[];
	    moreSpecific: string[];
	    routingSrc: Source;
	    registrySrc: Source;
	    rpkiSrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new PrefixReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.originAsn = source["originAsn"];
	        this.originName = source["originName"];
	        this.announced = source["announced"];
	        this.registry = source["registry"];
	        this.registeredOrg = source["registeredOrg"];
	        this.abuseEmail = source["abuseEmail"];
	        this.rpkiState = source["rpkiState"];
	        this.rpkiRoa = source["rpkiRoa"];
	        this.lessSpecific = source["lessSpecific"];
	        this.moreSpecific = source["moreSpecific"];
	        this.routingSrc = this.convertValues(source["routingSrc"], Source);
	        this.registrySrc = this.convertValues(source["registrySrc"], Source);
	        this.rpkiSrc = this.convertValues(source["rpkiSrc"], Source);
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

}

export namespace macvendor {
	
	export class Result {
	    input: string;
	    oui: string;
	    vendor: string;
	    found: boolean;
	    locallyAdministered: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.oui = source["oui"];
	        this.vendor = source["vendor"];
	        this.found = source["found"];
	        this.locallyAdministered = source["locallyAdministered"];
	    }
	}

}

export namespace netcat {
	
	export class CertInfo {
	    subject: string;
	    issuer: string;
	    notAfter: string;
	    verified: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CertInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subject = source["subject"];
	        this.issuer = source["issuer"];
	        this.notAfter = source["notAfter"];
	        this.verified = source["verified"];
	    }
	}

}

export namespace netflow {
	
	export class Conversation {
	    srcIp: string;
	    dstIp: string;
	    dstPort: number;
	    proto: string;
	    bytes: number;
	    packets: number;
	    flows: number;
	
	    static createFrom(source: any = {}) {
	        return new Conversation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.srcIp = source["srcIp"];
	        this.dstIp = source["dstIp"];
	        this.dstPort = source["dstPort"];
	        this.proto = source["proto"];
	        this.bytes = source["bytes"];
	        this.packets = source["packets"];
	        this.flows = source["flows"];
	    }
	}

}

export namespace netif {
	
	export class Iface {
	    index: number;
	    name: string;
	    mac: string;
	    mtu: number;
	    up: boolean;
	    loopback: boolean;
	    flags: string[];
	    ipv4: string[];
	    ipv6: string[];
	
	    static createFrom(source: any = {}) {
	        return new Iface(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	        this.mac = source["mac"];
	        this.mtu = source["mtu"];
	        this.up = source["up"];
	        this.loopback = source["loopback"];
	        this.flags = source["flags"];
	        this.ipv4 = source["ipv4"];
	        this.ipv6 = source["ipv6"];
	    }
	}

}

export namespace netinput {
	
	export class Result {
	    Kind: number;
	    Normalized: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Kind = source["Kind"];
	        this.Normalized = source["Normalized"];
	    }
	}

}

export namespace ntp {
	
	export class Reply {
	    server: string;
	    stratum: number;
	    refId: string;
	    leapIndicator: string;
	    precisionSeconds: number;
	    rootDelay: number;
	    rootDispersion: number;
	    // Go type: time
	    serverTime: any;
	    offset: number;
	    rtt: number;
	    kissOfDeath: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Reply(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.stratum = source["stratum"];
	        this.refId = source["refId"];
	        this.leapIndicator = source["leapIndicator"];
	        this.precisionSeconds = source["precisionSeconds"];
	        this.rootDelay = source["rootDelay"];
	        this.rootDispersion = source["rootDispersion"];
	        this.serverTime = this.convertValues(source["serverTime"], null);
	        this.offset = source["offset"];
	        this.rtt = source["rtt"];
	        this.kissOfDeath = source["kissOfDeath"];
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

}

export namespace ping {
	
	export class Reply {
	    seq: number;
	    rttMillis: number;
	    ok: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Reply(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.rttMillis = source["rttMillis"];
	        this.ok = source["ok"];
	    }
	}
	export class Stats {
	    host: string;
	    ip: string;
	    sent: number;
	    received: number;
	    lossPct: number;
	    minMs: number;
	    avgMs: number;
	    maxMs: number;
	    stddevMs: number;
	    jitterMs: number;
	    available: boolean;
	    replies: Reply[];
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.ip = source["ip"];
	        this.sent = source["sent"];
	        this.received = source["received"];
	        this.lossPct = source["lossPct"];
	        this.minMs = source["minMs"];
	        this.avgMs = source["avgMs"];
	        this.maxMs = source["maxMs"];
	        this.stddevMs = source["stddevMs"];
	        this.jitterMs = source["jitterMs"];
	        this.available = source["available"];
	        this.replies = this.convertValues(source["replies"], Reply);
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

}

export namespace pwaudit {
	
	export class Account {
	    name: string;
	    ntHash: string;
	    rid: string;
	    machine: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ntHash = source["ntHash"];
	        this.rid = source["rid"];
	        this.machine = source["machine"];
	    }
	}
	export class PatternStat {
	    pattern: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new PatternStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.count = source["count"];
	    }
	}
	export class ReuseGroup {
	    ntHash: string;
	    plaintext: string;
	    accounts: string[];
	
	    static createFrom(source: any = {}) {
	        return new ReuseGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ntHash = source["ntHash"];
	        this.plaintext = source["plaintext"];
	        this.accounts = source["accounts"];
	    }
	}
	export class Result {
	    account: Account;
	    cracked: boolean;
	    plaintext: string;
	    strength: string;
	    pattern: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account = this.convertValues(source["account"], Account);
	        this.cracked = source["cracked"];
	        this.plaintext = source["plaintext"];
	        this.strength = source["strength"];
	        this.pattern = source["pattern"];
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
	export class AuditReport {
	    total: number;
	    cracked: number;
	    percentCracked: number;
	    results: Result[];
	    reuse: ReuseGroup[];
	    patterns: PatternStat[];
	    crackerUsed: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.cracked = source["cracked"];
	        this.percentCracked = source["percentCracked"];
	        this.results = this.convertValues(source["results"], Result);
	        this.reuse = this.convertValues(source["reuse"], ReuseGroup);
	        this.patterns = this.convertValues(source["patterns"], PatternStat);
	        this.crackerUsed = source["crackerUsed"];
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
	export class CrackerInfo {
	    name: string;
	    version: string;
	    path: string;
	    available: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CrackerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.available = source["available"];
	    }
	}
	
	

}

export namespace scan {
	
	export class HostResult {
	    ip: string;
	    alive: boolean;
	    rtt: number;
	    method: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new HostResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.alive = source["alive"];
	        this.rtt = source["rtt"];
	        this.method = source["method"];
	        this.name = source["name"];
	    }
	}
	export class PortResult {
	    port: number;
	    state: string;
	    service: string;
	
	    static createFrom(source: any = {}) {
	        return new PortResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.state = source["state"];
	        this.service = source["service"];
	    }
	}

}

export namespace snmp {
	
	export class Interface {
	    index: number;
	    descr: string;
	    alias: string;
	    speedBps: number;
	    operStatus: string;
	
	    static createFrom(source: any = {}) {
	        return new Interface(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.descr = source["descr"];
	        this.alias = source["alias"];
	        this.speedBps = source["speedBps"];
	        this.operStatus = source["operStatus"];
	    }
	}
	export class MIBRow {
	    oid: string;
	    type: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new MIBRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oid = source["oid"];
	        this.type = source["type"];
	        this.value = source["value"];
	    }
	}
	export class PortEntry {
	    mac: string;
	    ip: string;
	    port: string;
	    ifIndex: number;
	    vlan: number;
	
	    static createFrom(source: any = {}) {
	        return new PortEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mac = source["mac"];
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.ifIndex = source["ifIndex"];
	        this.vlan = source["vlan"];
	    }
	}
	export class SweepHost {
	    ip: string;
	    responded: boolean;
	    name: string;
	    descr: string;
	    uptimeTicks: number;
	
	    static createFrom(source: any = {}) {
	        return new SweepHost(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.responded = source["responded"];
	        this.name = source["name"];
	        this.descr = source["descr"];
	        this.uptimeTicks = source["uptimeTicks"];
	    }
	}
	export class TrapVar {
	    oid: string;
	    type: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new TrapVar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oid = source["oid"];
	        this.type = source["type"];
	        this.value = source["value"];
	    }
	}
	export class TrapMsg {
	    // Go type: time
	    t: any;
	    source: string;
	    version: string;
	    trapOid: string;
	    trapName: string;
	    varbinds: TrapVar[];
	
	    static createFrom(source: any = {}) {
	        return new TrapMsg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.t = this.convertValues(source["t"], null);
	        this.source = source["source"];
	        this.version = source["version"];
	        this.trapOid = source["trapOid"];
	        this.trapName = source["trapName"];
	        this.varbinds = this.convertValues(source["varbinds"], TrapVar);
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

}

export namespace subnet {
	
	export class Info {
	    Prefix: string;
	    Version: number;
	    Network: string;
	    Broadcast: string;
	    Netmask: string;
	    Wildcard: string;
	    PrefixLen: number;
	    TotalCount: string;
	    UsableFirst: string;
	    UsableLast: string;
	    UsableCount: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Prefix = source["Prefix"];
	        this.Version = source["Version"];
	        this.Network = source["Network"];
	        this.Broadcast = source["Broadcast"];
	        this.Netmask = source["Netmask"];
	        this.Wildcard = source["Wildcard"];
	        this.PrefixLen = source["PrefixLen"];
	        this.TotalCount = source["TotalCount"];
	        this.UsableFirst = source["UsableFirst"];
	        this.UsableLast = source["UsableLast"];
	        this.UsableCount = source["UsableCount"];
	    }
	}

}

export namespace sysconf {
	
	export class Found {
	    values: string[];
	    source: string;
	    err: string;
	
	    static createFrom(source: any = {}) {
	        return new Found(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.values = source["values"];
	        this.source = source["source"];
	        this.err = source["err"];
	    }
	}

}

export namespace tfscan {
	
	export class Counts {
	    resources: number;
	    securityGroups: number;
	    exposed: number;
	    critical: number;
	    findings: number;
	
	    static createFrom(source: any = {}) {
	        return new Counts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resources = source["resources"];
	        this.securityGroups = source["securityGroups"];
	        this.exposed = source["exposed"];
	        this.critical = source["critical"];
	        this.findings = source["findings"];
	    }
	}
	export class ExposureEdge {
	    target: string;
	    label: string;
	    severity: string;
	
	    static createFrom(source: any = {}) {
	        return new ExposureEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.label = source["label"];
	        this.severity = source["severity"];
	    }
	}
	export class Finding {
	    severity: string;
	    category: string;
	    title: string;
	    detail: string;
	    resource: string;
	    file: string;
	    line: number;
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.resource = source["resource"];
	        this.file = source["file"];
	        this.line = source["line"];
	    }
	}
	export class Resource {
	    type: string;
	    name: string;
	    provider: string;
	    public: boolean;
	    exposure: string[];
	    dynamic: boolean;
	    file: string;
	    line: number;
	
	    static createFrom(source: any = {}) {
	        return new Resource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.public = source["public"];
	        this.exposure = source["exposure"];
	        this.dynamic = source["dynamic"];
	        this.file = source["file"];
	        this.line = source["line"];
	    }
	}
	export class Analysis {
	    source: string;
	    resources: Resource[];
	    findings: Finding[];
	    exposure: ExposureEdge[];
	    counts: Counts;
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new Analysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.resources = this.convertValues(source["resources"], Resource);
	        this.findings = this.convertValues(source["findings"], Finding);
	        this.exposure = this.convertValues(source["exposure"], ExposureEdge);
	        this.counts = this.convertValues(source["counts"], Counts);
	        this.notes = source["notes"];
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
	
	
	

}

export namespace tftp {
	
	export class File {
	    id: number;
	    name: string;
	    source: string;
	    size: number;
	    // Go type: time
	    t: any;
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.source = source["source"];
	        this.size = source["size"];
	        this.t = this.convertValues(source["t"], null);
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

}

export namespace tlscheck {
	
	export class CertReport {
	    host: string;
	    port: string;
	    subject: string;
	    issuer: string;
	    sans: string[];
	    serial: string;
	    // Go type: time
	    notBefore: any;
	    // Go type: time
	    notAfter: any;
	    daysToExpiry: number;
	    keyType: string;
	    keyBits: number;
	    sigAlg: string;
	    tlsVersion: string;
	    cipherSuite: string;
	    selfSigned: boolean;
	    chainTrusted: boolean;
	    hostnameMatch: boolean;
	    verdict: string;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new CertReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.subject = source["subject"];
	        this.issuer = source["issuer"];
	        this.sans = source["sans"];
	        this.serial = source["serial"];
	        this.notBefore = this.convertValues(source["notBefore"], null);
	        this.notAfter = this.convertValues(source["notAfter"], null);
	        this.daysToExpiry = source["daysToExpiry"];
	        this.keyType = source["keyType"];
	        this.keyBits = source["keyBits"];
	        this.sigAlg = source["sigAlg"];
	        this.tlsVersion = source["tlsVersion"];
	        this.cipherSuite = source["cipherSuite"];
	        this.selfSigned = source["selfSigned"];
	        this.chainTrusted = source["chainTrusted"];
	        this.hostnameMatch = source["hostnameMatch"];
	        this.verdict = source["verdict"];
	        this.warnings = source["warnings"];
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

}

export namespace trace {
	
	export class Hop {
	    ttl: number;
	    addr: string;
	    host: string;
	    rtts: number[];
	    asn: string;
	    asOwner: string;
	    timeout: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Hop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ttl = source["ttl"];
	        this.addr = source["addr"];
	        this.host = source["host"];
	        this.rtts = source["rtts"];
	        this.asn = source["asn"];
	        this.asOwner = source["asOwner"];
	        this.timeout = source["timeout"];
	    }
	}
	export class Result {
	    target: string;
	    addr: string;
	    hops: Hop[];
	    reached: boolean;
	    elapsed: number;
	    verdict: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.addr = source["addr"];
	        this.hops = this.convertValues(source["hops"], Hop);
	        this.reached = source["reached"];
	        this.elapsed = source["elapsed"];
	        this.verdict = source["verdict"];
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

}

export namespace update {
	
	export class Result {
	    current: string;
	    latest: string;
	    newer: boolean;
	    url: string;
	    notes: string;
	    downloadUrl: string;
	    assetName: string;
	    osLabel: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.newer = source["newer"];
	        this.url = source["url"];
	        this.notes = source["notes"];
	        this.downloadUrl = source["downloadUrl"];
	        this.assetName = source["assetName"];
	        this.osLabel = source["osLabel"];
	    }
	}

}

export namespace webconf {
	
	export class Counts {
	    servers: number;
	    listeners: number;
	    upstreams: number;
	    locations: number;
	    errors: number;
	    warnings: number;
	
	    static createFrom(source: any = {}) {
	        return new Counts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = source["servers"];
	        this.listeners = source["listeners"];
	        this.upstreams = source["upstreams"];
	        this.locations = source["locations"];
	        this.errors = source["errors"];
	        this.warnings = source["warnings"];
	    }
	}
	export class Finding {
	    severity: string;
	    category: string;
	    title: string;
	    detail: string;
	    file: string;
	    line: number;
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.file = source["file"];
	        this.line = source["line"];
	    }
	}
	export class Upstream {
	    name: string;
	    servers: string[];
	    policy: string;
	    usedBy: string[];
	    file: string;
	    line: number;
	
	    static createFrom(source: any = {}) {
	        return new Upstream(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.servers = source["servers"];
	        this.policy = source["policy"];
	        this.usedBy = source["usedBy"];
	        this.file = source["file"];
	        this.line = source["line"];
	    }
	}
	export class Location {
	    path: string;
	    kind: string;
	    target: string;
	    flags: string[];
	
	    static createFrom(source: any = {}) {
	        return new Location(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.target = source["target"];
	        this.flags = source["flags"];
	    }
	}
	export class Listener {
	    addr: string;
	    port: number;
	    ssl: boolean;
	    http2: boolean;
	    default: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Listener(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.addr = source["addr"];
	        this.port = source["port"];
	        this.ssl = source["ssl"];
	        this.http2 = source["http2"];
	        this.default = source["default"];
	    }
	}
	export class Server {
	    names: string[];
	    listeners: Listener[];
	    root: string;
	    locations: Location[];
	    tlsProtocols: string[];
	    tlsCert: string;
	    access: string[];
	    serverTokens: string;
	    headers: string[];
	    file: string;
	    line: number;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.names = source["names"];
	        this.listeners = this.convertValues(source["listeners"], Listener);
	        this.root = source["root"];
	        this.locations = this.convertValues(source["locations"], Location);
	        this.tlsProtocols = source["tlsProtocols"];
	        this.tlsCert = source["tlsCert"];
	        this.access = source["access"];
	        this.serverTokens = source["serverTokens"];
	        this.headers = source["headers"];
	        this.file = source["file"];
	        this.line = source["line"];
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
	export class Analysis {
	    serverType: string;
	    servers: Server[];
	    upstreams: Upstream[];
	    findings: Finding[];
	    counts: Counts;
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new Analysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverType = source["serverType"];
	        this.servers = this.convertValues(source["servers"], Server);
	        this.upstreams = this.convertValues(source["upstreams"], Upstream);
	        this.findings = this.convertValues(source["findings"], Finding);
	        this.counts = this.convertValues(source["counts"], Counts);
	        this.notes = source["notes"];
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
	
	
	
	
	

}

export namespace whois {
	
	export class Result {
	    object: string;
	    server: string;
	    chain: string[];
	    raw: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.object = source["object"];
	        this.server = source["server"];
	        this.chain = source["chain"];
	        this.raw = source["raw"];
	    }
	}

}

export namespace wordlist {
	
	export class Entry {
	    name: string;
	    desc: string;
	    url: string;
	    sha256: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.desc = source["desc"];
	        this.url = source["url"];
	        this.sha256 = source["sha256"];
	        this.size = source["size"];
	    }
	}

}

