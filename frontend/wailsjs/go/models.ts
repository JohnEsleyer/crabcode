export namespace main {
	
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

}

