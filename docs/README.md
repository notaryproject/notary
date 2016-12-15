<!--[metadata]>
+++
draft = true
+++
<![end-metadata]-->

# Contributing to the Docker Notary documentation

The documentation contained in this folder is for running a stand-alone notary service. For documentation on using Notary as part of Docker Content Trust,
please use the [Docker Documentation](https://docs.docker.com).

This documentation is built and hosted using Jekyll via GitHub Pages. The Dockerfile included in this directory can be used to build a docker image to run the 
docs in your local development environment.

Navigation is automatically generated based on the directory layout of this /docs folder and the title values of pages.

## Documentation contributing workflow

1. Edit or add a Markdown file in the tree.

2. Save your changes.

3. Make sure you are in the `docs` subdirectory.

4. Build the documentation.

        $ docker build -t notarydocs .
        $ docker run -it --rm -v $(pwd):/www -p 4000 notarydocs

5. Open your browser and navigate to [http://localhost:4000](http://localhost:4000)
