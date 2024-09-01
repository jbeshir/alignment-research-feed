ALTER TABLE `alignment_research_dataset`.`articles`
    ADD FULLTEXT INDEX `authors_fulltext_idx` (`authors`);
