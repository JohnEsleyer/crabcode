export namespace main {
	
	export class ButtonMappings {
	    run: string;
	    build: string;
	    test: string;
	
	    static createFrom(source: any = {}) {
	        return new ButtonMappings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run = source["run"];
	        this.build = source["build"];
	        this.test = source["test"];
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
	export class SetupStep {
	    name: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new SetupStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	    }
	}
	export class DeclarativeConfig {
	    name: string;
	    version: string;
	    environment: string;
	    iconColor: string;
	    setup: SetupStep[];
	    envVars: Record<string, string>;
	    mappings: ButtonMappings;
	    notes: NotesSpec;
	
	    static createFrom(source: any = {}) {
	        return new DeclarativeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.environment = source["environment"];
	        this.iconColor = source["iconColor"];
	        this.setup = this.convertValues(source["setup"], SetupStep);
	        this.envVars = source["envVars"];
	        this.mappings = this.convertValues(source["mappings"], ButtonMappings);
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
	
	export class Sandbox {
	    id: string;
	    workspaceId: string;
	    name: string;
	    folder: string;
	    markdownNote: string;
	    htmlNote: string;
	    isActive: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Sandbox(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workspaceId = source["workspaceId"];
	        this.name = source["name"];
	        this.folder = source["folder"];
	        this.markdownNote = source["markdownNote"];
	        this.htmlNote = source["htmlNote"];
	        this.isActive = source["isActive"];
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
	export class TemplateSpec {
	    id: string;
	    name: string;
	    environment: string;
	    iconColor: string;
	    config: DeclarativeConfig;
	    files: TemplateFile[];
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
	        this.files = this.convertValues(source["files"], TemplateFile);
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
	export class Workspace {
	    id: string;
	    name: string;
	    description: string;
	    configYaml: string;
	    runtimePath: string;
	    activeSandboxId: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Workspace(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.configYaml = source["configYaml"];
	        this.runtimePath = source["runtimePath"];
	        this.activeSandboxId = source["activeSandboxId"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

