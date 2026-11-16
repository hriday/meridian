export namespace acmecert {
	
	export class DNSRecord {
	    domain: string;
	    name: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new DNSRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.name = source["name"];
	        this.value = source["value"];
	    }
	}

}

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
	export class CLIInfo {
	    os: string;
	    available: boolean;
	    path: string;
	    installed: boolean;
	    linkPath: string;
	    onPath: boolean;
	    supported: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CLIInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.available = source["available"];
	        this.path = source["path"];
	        this.installed = source["installed"];
	        this.linkPath = source["linkPath"];
	        this.onPath = source["onPath"];
	        this.supported = source["supported"];
	    }
	}
	export class CertBeginResult {
	    orderId: string;
	    records: acmecert.DNSRecord[];
	    staging: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CertBeginResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.orderId = source["orderId"];
	        this.records = this.convertValues(source["records"], acmecert.DNSRecord);
	        this.staging = source["staging"];
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
	export class CertResult {
	    domains: string[];
	    fullchain: string;
	    key: string;
	
	    static createFrom(source: any = {}) {
	        return new CertResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domains = source["domains"];
	        this.fullchain = source["fullchain"];
	        this.key = source["key"];
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

export namespace arpscan {
	
	export class Host {
	    ip: string;
	    mac: string;
	    vendor?: string;
	
	    static createFrom(source: any = {}) {
	        return new Host(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.mac = source["mac"];
	        this.vendor = source["vendor"];
	    }
	}
	export class Result {
	    interface: string;
	    subnet: string;
	    hosts: Host[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.interface = source["interface"];
	        this.subnet = source["subnet"];
	        this.hosts = this.convertValues(source["hosts"], Host);
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

export namespace certexpiry {
	
	export class Result {
	    host: string;
	    ok: boolean;
	    error?: string;
	    subject?: string;
	    issuer?: string;
	    notAfter?: string;
	    daysLeft: number;
	    expired: boolean;
	    notYet: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.subject = source["subject"];
	        this.issuer = source["issuer"];
	        this.notAfter = source["notAfter"];
	        this.daysLeft = source["daysLeft"];
	        this.expired = source["expired"];
	        this.notYet = source["notYet"];
	    }
	}

}

export namespace certinfo {
	
	export class Item {
	    kind: string;
	    subject: string;
	    issuer?: string;
	    serialHex?: string;
	    // Go type: time
	    notBefore?: any;
	    // Go type: time
	    notAfter?: any;
	    daysLeft?: number;
	    expired?: boolean;
	    selfSigned?: boolean;
	    isCA?: boolean;
	    dnsNames?: string[];
	    ips?: string[];
	    emails?: string[];
	    uris?: string[];
	    keyType: string;
	    sigAlg: string;
	    keyUsage?: string[];
	    extKeyUsage?: string[];
	    version?: number;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.subject = source["subject"];
	        this.issuer = source["issuer"];
	        this.serialHex = source["serialHex"];
	        this.notBefore = this.convertValues(source["notBefore"], null);
	        this.notAfter = this.convertValues(source["notAfter"], null);
	        this.daysLeft = source["daysLeft"];
	        this.expired = source["expired"];
	        this.selfSigned = source["selfSigned"];
	        this.isCA = source["isCA"];
	        this.dnsNames = source["dnsNames"];
	        this.ips = source["ips"];
	        this.emails = source["emails"];
	        this.uris = source["uris"];
	        this.keyType = source["keyType"];
	        this.sigAlg = source["sigAlg"];
	        this.keyUsage = source["keyUsage"];
	        this.extKeyUsage = source["extKeyUsage"];
	        this.version = source["version"];
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
	export class Result {
	    items: Item[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Item);
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

export namespace cli {
	
	export class CommandDoc {
	    name: string;
	    args: string;
	    desc: string;
	
	    static createFrom(source: any = {}) {
	        return new CommandDoc(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.args = source["args"];
	        this.desc = source["desc"];
	    }
	}
	export class CommandGroup {
	    name: string;
	    note?: string;
	    commands: CommandDoc[];
	
	    static createFrom(source: any = {}) {
	        return new CommandGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.note = source["note"];
	        this.commands = this.convertValues(source["commands"], CommandDoc);
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
	export class RunbookStep {
	    command: string;
	    args: string[];
	    output: string;
	    ok: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RunbookStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.args = source["args"];
	        this.output = source["output"];
	        this.ok = source["ok"];
	    }
	}
	export class RunbookData {
	    profile: string;
	    name: string;
	    target: string;
	    generated: string;
	    steps: RunbookStep[];
	
	    static createFrom(source: any = {}) {
	        return new RunbookData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profile = source["profile"];
	        this.name = source["name"];
	        this.target = source["target"];
	        this.generated = source["generated"];
	        this.steps = this.convertValues(source["steps"], RunbookStep);
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
	export class RunbookInfo {
	    id: string;
	    name: string;
	    desc: string;
	    steps: number;
	
	    static createFrom(source: any = {}) {
	        return new RunbookInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.desc = source["desc"];
	        this.steps = source["steps"];
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

export namespace ctlog {
	
	export class Subdomain {
	    name: string;
	    wildcard: boolean;
	    certs: number;
	    firstSeen: string;
	    lastSeen: string;
	
	    static createFrom(source: any = {}) {
	        return new Subdomain(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.wildcard = source["wildcard"];
	        this.certs = source["certs"];
	        this.firstSeen = source["firstSeen"];
	        this.lastSeen = source["lastSeen"];
	    }
	}

}

export namespace cve {
	
	export class CVE {
	    id: string;
	    published: string;
	    description: string;
	    cvss: number;
	    severity: string;
	    vector: string;
	    cwe: string[];
	    kev: boolean;
	    references: string[];
	
	    static createFrom(source: any = {}) {
	        return new CVE(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.published = source["published"];
	        this.description = source["description"];
	        this.cvss = source["cvss"];
	        this.severity = source["severity"];
	        this.vector = source["vector"];
	        this.cwe = source["cwe"];
	        this.kev = source["kev"];
	        this.references = source["references"];
	    }
	}

}

export namespace dhcpscan {
	
	export class Server {
	    serverID: string;
	    offered: string;
	    subnetMask: string;
	    gateway: string;
	    dns: string[];
	    domainName: string;
	    leaseSecs: number;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverID = source["serverID"];
	        this.offered = source["offered"];
	        this.subnetMask = source["subnetMask"];
	        this.gateway = source["gateway"];
	        this.dns = source["dns"];
	        this.domainName = source["domainName"];
	        this.leaseSecs = source["leaseSecs"];
	    }
	}
	export class Result {
	    interface: string;
	    servers: Server[];
	    rogue: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.interface = source["interface"];
	        this.servers = this.convertValues(source["servers"], Server);
	        this.rogue = source["rogue"];
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

export namespace discover {
	
	export class Device {
	    address: string;
	    location: string;
	    server: string;
	    type: string;
	    usn: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.address = source["address"];
	        this.location = source["location"];
	        this.server = source["server"];
	        this.type = source["type"];
	        this.usn = source["usn"];
	        this.source = source["source"];
	    }
	}
	export class Service {
	    instance: string;
	    type: string;
	    host: string;
	    port: number;
	    addrs: string[];
	    info: string[];
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new Service(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.instance = source["instance"];
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.addrs = source["addrs"];
	        this.info = source["info"];
	        this.source = source["source"];
	    }
	}
	export class Result {
	    services: Service[];
	    devices: Device[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.services = this.convertValues(source["services"], Service);
	        this.devices = this.convertValues(source["devices"], Device);
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

export namespace dnsprop {
	
	export class Resolver {
	    name: string;
	    ip: string;
	    loc?: string;
	
	    static createFrom(source: any = {}) {
	        return new Resolver(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ip = source["ip"];
	        this.loc = source["loc"];
	    }
	}
	export class Answer {
	    resolver: Resolver;
	    values: string[];
	    ttl: number;
	    rcode: string;
	    rttMillis: number;
	    err?: string;
	    group: number;
	
	    static createFrom(source: any = {}) {
	        return new Answer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resolver = this.convertValues(source["resolver"], Resolver);
	        this.values = source["values"];
	        this.ttl = source["ttl"];
	        this.rcode = source["rcode"];
	        this.rttMillis = source["rttMillis"];
	        this.err = source["err"];
	        this.group = source["group"];
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
	export class Group {
	    values: string[];
	    rcode: string;
	    resolvers: string[];
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new Group(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.values = source["values"];
	        this.rcode = source["rcode"];
	        this.resolvers = source["resolvers"];
	        this.count = source["count"];
	    }
	}
	export class Report {
	    name: string;
	    type: string;
	    answers: Answer[];
	    groups: Group[];
	    consensus: boolean;
	    propagated: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.answers = this.convertValues(source["answers"], Answer);
	        this.groups = this.convertValues(source["groups"], Group);
	        this.consensus = source["consensus"];
	        this.propagated = source["propagated"];
	        this.total = source["total"];
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
	    transport: string;
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
	        this.transport = source["transport"];
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

export namespace emailauth {
	
	export class DKIMResult {
	    selector: string;
	    found: boolean;
	    record: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new DKIMResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selector = source["selector"];
	        this.found = source["found"];
	        this.record = source["record"];
	        this.note = source["note"];
	    }
	}
	export class DMARCResult {
	    status: string;
	    record: string;
	    policy: string;
	    pct: string;
	    rua: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new DMARCResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.record = source["record"];
	        this.policy = source["policy"];
	        this.pct = source["pct"];
	        this.rua = source["rua"];
	        this.note = source["note"];
	    }
	}
	export class SPFResult {
	    status: string;
	    record: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new SPFResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.record = source["record"];
	        this.note = source["note"];
	    }
	}
	export class Report {
	    domain: string;
	    spf: SPFResult;
	    dmarc: DMARCResult;
	    dkim: DKIMResult[];
	    dkimFound: boolean;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.spf = this.convertValues(source["spf"], SPFResult);
	        this.dmarc = this.convertValues(source["dmarc"], DMARCResult);
	        this.dkim = this.convertValues(source["dkim"], DKIMResult);
	        this.dkimFound = source["dkimFound"];
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

export namespace h3probe {
	
	export class Result {
	    host: string;
	    port: number;
	    quicReachable: boolean;
	    alpn: string;
	    tlsVersion: string;
	    handshakeMillis: number;
	    advertisesH3: boolean;
	    altSvc: string;
	    quicError?: string;
	    verdict: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.quicReachable = source["quicReachable"];
	        this.alpn = source["alpn"];
	        this.tlsVersion = source["tlsVersion"];
	        this.handshakeMillis = source["handshakeMillis"];
	        this.advertisesH3 = source["advertisesH3"];
	        this.altSvc = source["altSvc"];
	        this.quicError = source["quicError"];
	        this.verdict = source["verdict"];
	    }
	}

}

export namespace hashid {
	
	export class Candidate {
	    name: string;
	    mode: number;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.mode = source["mode"];
	        this.note = source["note"];
	    }
	}

}

export namespace hibp {
	
	export class Breach {
	    name: string;
	    title: string;
	    domain: string;
	    breachDate: string;
	    pwnCount: number;
	    description: string;
	    dataClasses: string[];
	    verified: boolean;
	    sensitive: boolean;
	    malware: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Breach(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.domain = source["domain"];
	        this.breachDate = source["breachDate"];
	        this.pwnCount = source["pwnCount"];
	        this.description = source["description"];
	        this.dataClasses = source["dataClasses"];
	        this.verified = source["verified"];
	        this.sensitive = source["sensitive"];
	        this.malware = source["malware"];
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

export namespace ikeprobe {
	
	export class Transform {
	    kind: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Transform(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.name = source["name"];
	    }
	}
	export class Result {
	    target: string;
	    responds: boolean;
	    version?: string;
	    accepted?: Transform[];
	    natTraversal: boolean;
	    notify?: string[];
	    rttMs?: number;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.responds = source["responds"];
	        this.version = source["version"];
	        this.accepted = this.convertValues(source["accepted"], Transform);
	        this.natTraversal = source["natTraversal"];
	        this.notify = source["notify"];
	        this.rttMs = source["rttMs"];
	        this.note = source["note"];
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
	    utilPct: number;
	
	    static createFrom(source: any = {}) {
	        return new TreeNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.record = this.convertValues(source["record"], Record);
	        this.children = this.convertValues(source["children"], TreeNode);
	        this.utilPct = source["utilPct"];
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

export namespace jwtinspect {
	
	export class Decoded {
	    headerJSON: string;
	    payloadJSON: string;
	    alg: string;
	    typ?: string;
	    kid?: string;
	    issuer?: string;
	    subject?: string;
	    audience?: string[];
	    jwtID?: string;
	    // Go type: time
	    issuedAt?: any;
	    // Go type: time
	    notBefore?: any;
	    // Go type: time
	    expiresAt?: any;
	    expired: boolean;
	    notYetValid: boolean;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Decoded(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.headerJSON = source["headerJSON"];
	        this.payloadJSON = source["payloadJSON"];
	        this.alg = source["alg"];
	        this.typ = source["typ"];
	        this.kid = source["kid"];
	        this.issuer = source["issuer"];
	        this.subject = source["subject"];
	        this.audience = source["audience"];
	        this.jwtID = source["jwtID"];
	        this.issuedAt = this.convertValues(source["issuedAt"], null);
	        this.notBefore = this.convertValues(source["notBefore"], null);
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
	        this.expired = source["expired"];
	        this.notYetValid = source["notYetValid"];
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
	export class VerifyResult {
	    algorithm: string;
	    valid: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new VerifyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.algorithm = source["algorithm"];
	        this.valid = source["valid"];
	        this.reason = source["reason"];
	    }
	}

}

export namespace lanrecon {
	
	export class ApplyResult {
	    imported: number;
	    updated: number;
	    deprecated: number;
	
	    static createFrom(source: any = {}) {
	        return new ApplyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imported = source["imported"];
	        this.updated = source["updated"];
	        this.deprecated = source["deprecated"];
	    }
	}
	export class Finding {
	    ip: string;
	    mac: string;
	    vendor?: string;
	    category: string;
	    recordId?: string;
	    name?: string;
	    knownMac?: string;
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.mac = source["mac"];
	        this.vendor = source["vendor"];
	        this.category = source["category"];
	        this.recordId = source["recordId"];
	        this.name = source["name"];
	        this.knownMac = source["knownMac"];
	    }
	}
	export class Report {
	    subnet: string;
	    findings: Finding[];
	    new: number;
	    inSync: number;
	    macChange: number;
	    missing: number;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subnet = source["subnet"];
	        this.findings = this.convertValues(source["findings"], Finding);
	        this.new = source["new"];
	        this.inSync = source["inSync"];
	        this.macChange = source["macChange"];
	        this.missing = source["missing"];
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

export namespace lldp {
	
	export class Neighbor {
	    protocol: string;
	    systemName: string;
	    chassisID?: string;
	    portID: string;
	    portDesc?: string;
	    systemDesc?: string;
	    platform?: string;
	    mgmtIP?: string;
	    vlan?: number;
	    capabilities?: string[];
	    sourceMAC: string;
	
	    static createFrom(source: any = {}) {
	        return new Neighbor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.protocol = source["protocol"];
	        this.systemName = source["systemName"];
	        this.chassisID = source["chassisID"];
	        this.portID = source["portID"];
	        this.portDesc = source["portDesc"];
	        this.systemDesc = source["systemDesc"];
	        this.platform = source["platform"];
	        this.mgmtIP = source["mgmtIP"];
	        this.vlan = source["vlan"];
	        this.capabilities = source["capabilities"];
	        this.sourceMAC = source["sourceMAC"];
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
	    prefixes?: string[];
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
	        this.prefixes = source["prefixes"];
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
	export class IRRRoute {
	    prefix: string;
	    origin: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new IRRRoute(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prefix = source["prefix"];
	        this.origin = source["origin"];
	        this.source = source["source"];
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
	    irrState: string;
	    irrRoutes: IRRRoute[];
	    vantages: BGPCollector[];
	    routingSrc: Source;
	    rpkiSrc: Source;
	    irrSrc: Source;
	
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
	        this.irrState = source["irrState"];
	        this.irrRoutes = this.convertValues(source["irrRoutes"], IRRRoute);
	        this.vantages = this.convertValues(source["vantages"], BGPCollector);
	        this.routingSrc = this.convertValues(source["routingSrc"], Source);
	        this.rpkiSrc = this.convertValues(source["rpkiSrc"], Source);
	        this.irrSrc = this.convertValues(source["irrSrc"], Source);
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

export namespace mtu {
	
	export class Result {
	    target: string;
	    addr: string;
	    dfEnforced: boolean;
	    note?: string;
	    pathMTU: number;
	    payloadMax: number;
	    localMTU: number;
	    nextHopMTU?: number;
	    bottleneck: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.addr = source["addr"];
	        this.dfEnforced = source["dfEnforced"];
	        this.note = source["note"];
	        this.pathMTU = source["pathMTU"];
	        this.payloadMax = source["payloadMax"];
	        this.localMTU = source["localMTU"];
	        this.nextHopMTU = source["nextHopMTU"];
	        this.bottleneck = source["bottleneck"];
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

export namespace pktscan {
	
	export class DNSError {
	    rcode: number;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new DNSError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rcode = source["rcode"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class ProtoStat {
	    proto: string;
	    packets: number;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new ProtoStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proto = source["proto"];
	        this.packets = source["packets"];
	        this.bytes = source["bytes"];
	    }
	}
	export class Talker {
	    ip: string;
	    packets: number;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new Talker(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.packets = source["packets"];
	        this.bytes = source["bytes"];
	    }
	}
	export class Summary {
	    interface: string;
	    seconds: number;
	    packets: number;
	    bytes: number;
	    topTalkers: Talker[];
	    protocols: ProtoStat[];
	    tcpSyns: number;
	    tcpResets: number;
	    retransmits: number;
	    dnsErrors: DNSError[];
	    snis: string[];
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.interface = source["interface"];
	        this.seconds = source["seconds"];
	        this.packets = source["packets"];
	        this.bytes = source["bytes"];
	        this.topTalkers = this.convertValues(source["topTalkers"], Talker);
	        this.protocols = this.convertValues(source["protocols"], ProtoStat);
	        this.tcpSyns = source["tcpSyns"];
	        this.tcpResets = source["tcpResets"];
	        this.retransmits = source["retransmits"];
	        this.dnsErrors = this.convertValues(source["dnsErrors"], DNSError);
	        this.snis = source["snis"];
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

export namespace ptrsweep {
	
	export class Entry {
	    ip: string;
	    names: string[];
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.names = source["names"];
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

export namespace pwhash {
	
	export class CheckResult {
	    match: boolean;
	    cost?: number;
	
	    static createFrom(source: any = {}) {
	        return new CheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.match = source["match"];
	        this.cost = source["cost"];
	    }
	}
	export class GenResult {
	    hash: string;
	    cost: number;
	    htpasswd?: string;
	
	    static createFrom(source: any = {}) {
	        return new GenResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.cost = source["cost"];
	        this.htpasswd = source["htpasswd"];
	    }
	}

}

export namespace rtp {
	
	export class Stream {
	    ssrc: number;
	    src: string;
	    dst: string;
	    payloadType: number;
	    codec: string;
	    packets: number;
	    lost: number;
	    lossPct: number;
	    jitterMs: number;
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new Stream(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ssrc = source["ssrc"];
	        this.src = source["src"];
	        this.dst = source["dst"];
	        this.payloadType = source["payloadType"];
	        this.codec = source["codec"];
	        this.packets = source["packets"];
	        this.lost = source["lost"];
	        this.lossPct = source["lossPct"];
	        this.jitterMs = source["jitterMs"];
	        this.durationMs = source["durationMs"];
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

export namespace secheaders {
	
	export class Header {
	    name: string;
	    present: boolean;
	    value: string;
	    grade: string;
	    note: string;
	    owasp: string;
	
	    static createFrom(source: any = {}) {
	        return new Header(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.present = source["present"];
	        this.value = source["value"];
	        this.grade = source["grade"];
	        this.note = source["note"];
	        this.owasp = source["owasp"];
	    }
	}
	export class Report {
	    url: string;
	    finalUrl: string;
	    https: boolean;
	    status: number;
	    headers: Header[];
	    score: number;
	    grade: string;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.finalUrl = source["finalUrl"];
	        this.https = source["https"];
	        this.status = source["status"];
	        this.headers = this.convertValues(source["headers"], Header);
	        this.score = source["score"];
	        this.grade = source["grade"];
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

export namespace sipping {
	
	export class Result {
	    target: string;
	    transport: string;
	    statusCode: number;
	    reason: string;
	    server?: string;
	    userAgent?: string;
	    allow?: string[];
	    accept?: string[];
	    supported?: string[];
	    rttMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.transport = source["transport"];
	        this.statusCode = source["statusCode"];
	        this.reason = source["reason"];
	        this.server = source["server"];
	        this.userAgent = source["userAgent"];
	        this.allow = source["allow"];
	        this.accept = source["accept"];
	        this.supported = source["supported"];
	        this.rttMillis = source["rttMillis"];
	    }
	}

}

export namespace smtpprobe {
	
	export class TLSInfo {
	    version: string;
	    cipher: string;
	    subject: string;
	    issuer: string;
	    dnsNames: string[];
	    // Go type: time
	    notAfter: any;
	    daysLeft: number;
	    hostMatch: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TLSInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.cipher = source["cipher"];
	        this.subject = source["subject"];
	        this.issuer = source["issuer"];
	        this.dnsNames = source["dnsNames"];
	        this.notAfter = this.convertValues(source["notAfter"], null);
	        this.daysLeft = source["daysLeft"];
	        this.hostMatch = source["hostMatch"];
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
	export class Result {
	    host: string;
	    port: string;
	    banner: string;
	    greeting: string;
	    capabilities: string[];
	    startTLS: boolean;
	    tls?: TLSInfo;
	    implicitTLS: boolean;
	    authMechs?: string[];
	    rttMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.banner = source["banner"];
	        this.greeting = source["greeting"];
	        this.capabilities = source["capabilities"];
	        this.startTLS = source["startTLS"];
	        this.tls = this.convertValues(source["tls"], TLSInfo);
	        this.implicitTLS = source["implicitTLS"];
	        this.authMechs = source["authMechs"];
	        this.rttMillis = source["rttMillis"];
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

export namespace sshaudit {
	
	export class HostKey {
	    type: string;
	    bits: number;
	    fingerprint: string;
	
	    static createFrom(source: any = {}) {
	        return new HostKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.bits = source["bits"];
	        this.fingerprint = source["fingerprint"];
	    }
	}
	export class PQStatus {
	    supported: boolean;
	    preferred: boolean;
	    algorithms: string[];
	    verdict: string;
	
	    static createFrom(source: any = {}) {
	        return new PQStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.preferred = source["preferred"];
	        this.algorithms = source["algorithms"];
	        this.verdict = source["verdict"];
	    }
	}
	export class Report {
	    host: string;
	    serverId: string;
	    software: string;
	    kex: string[];
	    hostKeyAlgos: string[];
	    ciphers: string[];
	    macs: string[];
	    hostKeys: HostKey[];
	    pq: PQStatus;
	    grade: string;
	    findings: string[];
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.serverId = source["serverId"];
	        this.software = source["software"];
	        this.kex = source["kex"];
	        this.hostKeyAlgos = source["hostKeyAlgos"];
	        this.ciphers = source["ciphers"];
	        this.macs = source["macs"];
	        this.hostKeys = this.convertValues(source["hostKeys"], HostKey);
	        this.pq = this.convertValues(source["pq"], PQStatus);
	        this.grade = source["grade"];
	        this.findings = source["findings"];
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

export namespace stun {
	
	export class Result {
	    server: string;
	    localIP: string;
	    publicIP: string;
	    publicPort: number;
	    nat: boolean;
	    natType: string;
	    rttMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.localIP = source["localIP"];
	        this.publicIP = source["publicIP"];
	        this.publicPort = source["publicPort"];
	        this.nat = source["nat"];
	        this.natType = source["natType"];
	        this.rttMillis = source["rttMillis"];
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

export namespace svcstatus {
	
	export class Component {
	    name: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Component(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	    }
	}
	export class Incident {
	    name: string;
	    status: string;
	    impact: string;
	    shortlink: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Incident(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.impact = source["impact"];
	        this.shortlink = source["shortlink"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Status {
	    service: string;
	    url: string;
	    indicator: string;
	    description: string;
	    operational: boolean;
	    impacted: Component[];
	    incidents: Incident[];
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.service = source["service"];
	        this.url = source["url"];
	        this.indicator = source["indicator"];
	        this.description = source["description"];
	        this.operational = source["operational"];
	        this.impacted = this.convertValues(source["impacted"], Component);
	        this.incidents = this.convertValues(source["incidents"], Incident);
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

export namespace tcpping {
	
	export class Probe {
	    seq: number;
	    rttMs: number;
	    ok: boolean;
	    err?: string;
	
	    static createFrom(source: any = {}) {
	        return new Probe(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.rttMs = source["rttMs"];
	        this.ok = source["ok"];
	        this.err = source["err"];
	    }
	}
	export class Result {
	    host: string;
	    port: number;
	    address: string;
	    sent: number;
	    received: number;
	    lossPct: number;
	    minMs: number;
	    avgMs: number;
	    maxMs: number;
	    jitterMs: number;
	    verdict: string;
	    probes: Probe[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.address = source["address"];
	        this.sent = source["sent"];
	        this.received = source["received"];
	        this.lossPct = source["lossPct"];
	        this.minMs = source["minMs"];
	        this.avgMs = source["avgMs"];
	        this.maxMs = source["maxMs"];
	        this.jitterMs = source["jitterMs"];
	        this.verdict = source["verdict"];
	        this.probes = this.convertValues(source["probes"], Probe);
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

export namespace threatintel {
	
	export class Hit {
	    ip: string;
	    listed: boolean;
	    malware?: string;
	    status?: string;
	    country?: string;
	    asNumber?: number;
	    asName?: string;
	    port?: number;
	    firstSeen?: string;
	    lastOnline?: string;
	
	    static createFrom(source: any = {}) {
	        return new Hit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.listed = source["listed"];
	        this.malware = source["malware"];
	        this.status = source["status"];
	        this.country = source["country"];
	        this.asNumber = source["asNumber"];
	        this.asName = source["asName"];
	        this.port = source["port"];
	        this.firstSeen = source["firstSeen"];
	        this.lastOnline = source["lastOnline"];
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

export namespace tlsscan {
	
	export class Cipher {
	    id: number;
	    name: string;
	    secure: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new Cipher(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.secure = source["secure"];
	        this.version = source["version"];
	    }
	}
	export class PQStatus {
	    supported: boolean;
	    group: string;
	    verdict: string;
	
	    static createFrom(source: any = {}) {
	        return new PQStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.group = source["group"];
	        this.verdict = source["verdict"];
	    }
	}
	export class Protocol {
	    name: string;
	    version: number;
	    supported: boolean;
	    legacy: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Protocol(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.supported = source["supported"];
	        this.legacy = source["legacy"];
	    }
	}
	export class Report {
	    host: string;
	    protocols: Protocol[];
	    ciphers: Cipher[];
	    pq: PQStatus;
	    certSignature: string;
	    grade: string;
	    findings: string[];
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.protocols = this.convertValues(source["protocols"], Protocol);
	        this.ciphers = this.convertValues(source["ciphers"], Cipher);
	        this.pq = this.convertValues(source["pq"], PQStatus);
	        this.certSignature = source["certSignature"];
	        this.grade = source["grade"];
	        this.findings = source["findings"];
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

export namespace totp {
	
	export class Code {
	    code: string;
	    secondsLeft: number;
	    period: number;
	    issuer?: string;
	    account?: string;
	
	    static createFrom(source: any = {}) {
	        return new Code(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.secondsLeft = source["secondsLeft"];
	        this.period = source["period"];
	        this.issuer = source["issuer"];
	        this.account = source["account"];
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
	export class NetInfo {
	    asn: string;
	    owner: string;
	    policy: string;
	    ratio: string;
	    ixCount: number;
	    facCount: number;
	    ixps: string[];
	    irrAsSet: string;
	
	    static createFrom(source: any = {}) {
	        return new NetInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.asn = source["asn"];
	        this.owner = source["owner"];
	        this.policy = source["policy"];
	        this.ratio = source["ratio"];
	        this.ixCount = source["ixCount"];
	        this.facCount = source["facCount"];
	        this.ixps = source["ixps"];
	        this.irrAsSet = source["irrAsSet"];
	    }
	}
	export class Result {
	    target: string;
	    addr: string;
	    hops: Hop[];
	    reached: boolean;
	    elapsed: number;
	    verdict: string;
	    networks?: NetInfo[];
	
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
	        this.networks = this.convertValues(source["networks"], NetInfo);
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

export namespace transcode {
	
	export class Guess {
	    label: string;
	    result: string;
	
	    static createFrom(source: any = {}) {
	        return new Guess(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.result = source["result"];
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

export namespace voipcalc {
	
	export class Codec {
	    name: string;
	    bitrateKbps: number;
	    defaultPtimeMs: number;
	
	    static createFrom(source: any = {}) {
	        return new Codec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.bitrateKbps = source["bitrateKbps"];
	        this.defaultPtimeMs = source["defaultPtimeMs"];
	    }
	}
	export class Result {
	    codec: string;
	    ptimeMs: number;
	    payloadBytes: number;
	    packetsPerSec: number;
	    oneWayKbps: number;
	    oneWayEthKbps: number;
	    perCallKbps: number;
	    calls: number;
	    totalKbps: number;
	    totalEthKbps: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.codec = source["codec"];
	        this.ptimeMs = source["ptimeMs"];
	        this.payloadBytes = source["payloadBytes"];
	        this.packetsPerSec = source["packetsPerSec"];
	        this.oneWayKbps = source["oneWayKbps"];
	        this.oneWayEthKbps = source["oneWayEthKbps"];
	        this.perCallKbps = source["perCallKbps"];
	        this.calls = source["calls"];
	        this.totalKbps = source["totalKbps"];
	        this.totalEthKbps = source["totalEthKbps"];
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

export namespace wgprobe {
	
	export class Result {
	    target: string;
	    validMac1: boolean;
	    responds: boolean;
	    replyType?: string;
	    rttMs?: number;
	    verdict: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.validMac1 = source["validMac1"];
	        this.responds = source["responds"];
	        this.replyType = source["replyType"];
	        this.rttMs = source["rttMs"];
	        this.verdict = source["verdict"];
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

export namespace wifi {
	
	export class Network {
	    ssid: string;
	    channel: number;
	    band: string;
	    width: string;
	    phyMode: string;
	    security: string;
	    rssi: number;
	    noise: number;
	    snr: number;
	    txRate?: number;
	    coChannel: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Network(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ssid = source["ssid"];
	        this.channel = source["channel"];
	        this.band = source["band"];
	        this.width = source["width"];
	        this.phyMode = source["phyMode"];
	        this.security = source["security"];
	        this.rssi = source["rssi"];
	        this.noise = source["noise"];
	        this.snr = source["snr"];
	        this.txRate = source["txRate"];
	        this.coChannel = source["coChannel"];
	    }
	}
	export class Info {
	    interface: string;
	    connected: boolean;
	    current: Network;
	    neighbors: Network[];
	    coChannel: number;
	    ssidRedacted: boolean;
	    locationStatus: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.interface = source["interface"];
	        this.connected = source["connected"];
	        this.current = this.convertValues(source["current"], Network);
	        this.neighbors = this.convertValues(source["neighbors"], Network);
	        this.coChannel = source["coChannel"];
	        this.ssidRedacted = source["ssidRedacted"];
	        this.locationStatus = source["locationStatus"];
	        this.note = source["note"];
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

