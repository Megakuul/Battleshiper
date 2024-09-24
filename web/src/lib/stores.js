import { writable } from "svelte/store";

/** @type {import("svelte/store").Writable<import("$lib/adapter/user/fetchinfo").fetchInfoOutput|undefined>} */
export const UserInfo = writable(undefined); 

/** @type {import("svelte/store").Writable<import("$lib/adapter/resource/listrepository").listRepositoryOutput|undefined>} */
export const RepositoryInfo = writable({
  message: "Repository list retrieved successfully",
  repositories: [
    {
      id: 1296269,
      name: "Hello-World",
      full_name: "octocat/Hello-World"
    },
    {
      id: 5434674,
      name: "Spoon-Knife",
      full_name: "octocat/Spoon-Knife"
    }
  ]
});

/** @type {import("svelte/store").Writable<import("$lib/adapter/resource/listproject").listProjectOutput|undefined>} */
export const ProjectInfo = writable({
  message: "Project list retrieved successfully",
  projects: [
    {
      name: "example",
      deleted: false,
      initialized: true,
      status: "DEPLOYMENT FAILED: BLDIJSFOISJDFO JEOJF OIEJSOFJOWWWJWO EJOIWJOFJWOIJEOWEJF OWJEFOJWEFOJWEO JWEO",
      build_image: "node:14-alpine",
      build_command: "npm run build",
      output_directory: "/dist",
      repository: {
        id: 1,
        url: "https://github.com/example/repo",
        branch: "main"
      },
      aliases: {
        prod: {
          url: "https://example.com",
          active: true
        },
        staging: {
          url: "https://staging.example.com",
          active: false
        }
      },
      last_event_result: {
        execution_identifier: "7e8237e9-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1632850400000,
        successful: true
      },
      last_build_result: {
        execution_identifier: "7e8237e9-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1632850500000,
        successful: false
      },
      last_deployment_result: {
        execution_identifier: "7e8237e9-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1632850600000,
        successful: true
      }
    },
    {
      name: "project2",
      deleted: false,
      initialized: true,
      status: "",
      build_image: "node:16-alpine",
      build_command: "npm run build",
      output_directory: "/build",
      repository: {
        id: 2,
        url: "https://github.com/example2/repo",
        branch: "develop"
      },
      aliases: {
        prod: {
          url: "https://prod.example2.com",
          active: true
        },
        staging: {
          url: "https://staging.example2.com",
          active: false
        }
      },
      last_event_result: {
        execution_identifier: "12345678-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1632950400000,
        successful: true
      },
      last_build_result: {
        execution_identifier: "12345678-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1632950500000,
        successful: true
      },
      last_deployment_result: {
        execution_identifier: "12345678-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1632950600000,
        successful: true
      }
    },
    {
      name: "project3",
      deleted: true,
      initialized: false,
      status: "Project archived",
      build_image: "node:12-alpine",
      build_command: "npm run build",
      output_directory: "/output",
      repository: {
        id: 3,
        url: "https://github.com/example3/repo",
        branch: "main"
      },
      aliases: {
        prod: {
          url: "https://prod.example3.com",
          active: false
        },
        staging: {
          url: "https://staging.example3.com",
          active: false
        }
      },
      last_event_result: {
        execution_identifier: "abcdef12-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1633050400000,
        successful: false
      },
      last_build_result: {
        execution_identifier: "abcdef12-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1633050500000,
        successful: false
      },
      last_deployment_result: {
        execution_identifier: "abcdef12-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1633050600000,
        successful: false
      }
    },
    {
      name: "project4",
      deleted: false,
      initialized: true,
      status: "BUILD IN PROGRESS",
      build_image: "node:18-alpine",
      build_command: "npm run build",
      output_directory: "/dist",
      repository: {
        id: 4,
        url: "https://github.com/example4/repo",
        branch: "feature-branch"
      },
      aliases: {
        prod: {
          url: "https://prod.example4.com",
          active: false
        },
        staging: {
          url: "https://staging.example4.com",
          active: true
        }
      },
      last_event_result: {
        execution_identifier: "1234abcd-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1633150400000,
        successful: true
      },
      last_build_result: {
        execution_identifier: "1234abcd-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1633150500000,
        successful: true
      },
      last_deployment_result: {
        execution_identifier: "1234abcd-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1633150600000,
        successful: false
      }
    },
    {
      name: "project5",
      deleted: false,
      initialized: true,
      status: "DEPLOYMENT PENDING",
      build_image: "python:3.9-slim",
      build_command: "python setup.py build",
      output_directory: "/output",
      repository: {
        id: 5,
        url: "https://github.com/example5/repo",
        branch: "main"
      },
      aliases: {
        prod: {
          url: "https://prod.example5.com",
          active: false
        },
        staging: {
          url: "https://staging.example5.com",
          active: true
        }
      },
      last_event_result: {
        execution_identifier: "9abcd123-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1633250400000,
        successful: true
      },
      last_build_result: {
        execution_identifier: "9abcd123-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1633250500000,
        successful: false
      },
      last_deployment_result: {
        execution_identifier: "9abcd123-ea45-4180-822f-2a104c9e12e0",
        timestamp: 1633250600000,
        successful: false
      }
    }
  ]
}); 