<!--[metadata]>
+++
draft=true
title = "Overview of Docker Notary"
description = "Overview of Docker Notary"
keywords = ["docker, notary, trust, image, signing, repository"]
[menu.main]
parent="mn_notary"
weight=-99
+++
<![end-metadata]-->

# Overview of Docker Notary

Notary is a tool for publishing and managing trusted collections of content. Publishers can digitally sign collections and consumers can verify integrity and origin of content. This ability is built on a straightforward key management and signing interface to create signed collections and configure trusted publishers.

With notary anyone can provide trust over arbitrary collections of data. Using The Update Framework (TUF) as the underlying security framework, it takes care of the operations necessary to create, manage and distribute the metadata necessary to ensure the integrity and freshness of your content.

