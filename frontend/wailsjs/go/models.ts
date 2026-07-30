export namespace main {
	
	export class BuildStep {
	    name: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	    }
	}
	export class NotesSpec {
	    markdown: string;
	    html: string;
	
	    static createFrom(source: any = {}) {
	        return new NotesSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.markdown = source["markdown"];
	        this.html = source["html"];
	    }
	}
	export class RunSpec {
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new RunSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	    }
	}
	export class TemplateFile {
	    path: string;
	    content: string;
	    isDir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TemplateFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.content = source["content"];
	        this.isDir = source["isDir"];
	    }
	}
	export class SetupStep {
	    name: string;
	    command: string;
	    dir: string;
	
	    static createFrom(source: any = {}) {
	        return new SetupStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	        this.dir = source["dir"];
	    }
	}
	export class DeclarativeConfig {
	    name: string;
	    version: string;
	    environment: string;
	    envDir: string;
	    iconColor: string;
	    setup: SetupStep[];
	    envVars: Record<string, string>;
	    files: TemplateFile[];
	    build: BuildStep[];
	    run: RunSpec;
	    notes: NotesSpec;
	
	    static createFrom(source: any = {}) {
	        return new DeclarativeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.environment = source["environment"];
	        this.envDir = source["envDir"];
	        this.iconColor = source["iconColor"];
	        this.setup = this.convertValues(source["setup"], SetupStep);
	        this.envVars = source["envVars"];
	        this.files = this.convertValues(source["files"], TemplateFile);
	        this.build = this.convertValues(source["build"], BuildStep);
	        this.run = this.convertValues(source["run"], RunSpec);
	        this.notes = this.convertValues(source["notes"], NotesSpec);
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
	export class FileNode {
	    name: string;
	    path: string;
	    isDir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	    }
	}
	export class GlobalSettings {
	    crabRootPath: string;
	    universalEnvDir: string;
	
	    static createFrom(source: any = {}) {
	        return new GlobalSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.crabRootPath = source["crabRootPath"];
	        this.universalEnvDir = source["universalEnvDir"];
	    }
	}
	export class Note {
	    id: string;
	    title: string;
	    content: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Note(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	
	
	export class Sandbox {
	    id: string;
	    name: string;
	    configYaml: string;
	    markdownNote: string;
	    htmlNote: string;
	    folder: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Sandbox(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.configYaml = source["configYaml"];
	        this.markdownNote = source["markdownNote"];
	        this.htmlNote = source["htmlNote"];
	        this.folder = source["folder"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SandboxFile {
	    id: string;
	    sandboxId: string;
	    path: string;
	    content: string;
	    isDir: boolean;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SandboxFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sandboxId = source["sandboxId"];
	        this.path = source["path"];
	        this.content = source["content"];
	        this.isDir = source["isDir"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	
	
	export class TemplateSpec {
	    id: string;
	    name: string;
	    environment: string;
	    iconColor: string;
	    config: DeclarativeConfig;
	    rawYaml: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplateSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.environment = source["environment"];
	        this.iconColor = source["iconColor"];
	        this.config = this.convertValues(source["config"], DeclarativeConfig);
	        this.rawYaml = source["rawYaml"];
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
	export class WorkspaceInfo {
	    path: string;
	    notes: Note[];
	    sandboxes: Sandbox[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.notes = this.convertValues(source["notes"], Note);
	        this.sandboxes = this.convertValues(source["sandboxes"], Sandbox);
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
	export class WorkspaceInitInfo {
	    path: string;
	    hasDotCrab: boolean;
	    exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceInitInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.hasDotCrab = source["hasDotCrab"];
	        this.exists = source["exists"];
	    }
	}

}

